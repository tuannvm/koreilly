package oreilly

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// Chapter represents a chapter/section in an O'Reilly book.
type Chapter struct {
	Title string
	URL   string
}

// FetchTOC tries to fetch a Table of Contents (ToC) for a given book slug.
// Returns an array of chapters/sections (title and URL path), or an error.
// This just prints each chapter URL for now as a proof-of-concept.
func (s *Service) FetchTOC(ctx context.Context, jwt, slug, bookID string) ([]Chapter, error) {
	if jwt == "" {
		return nil, fmt.Errorf("empty JWT")
	}
	if slug == "" || bookID == "" {
		return nil, fmt.Errorf("empty slug or book ID")
	}

	// Try an endpoint for known/modern O'Reilly: /api/v2/library/{slug}/toc/
	apiURL := fmt.Sprintf("https://learning.oreilly.com/api/v2/library/%s/toc/", slug)
	fmt.Printf("[oreilly][FetchTOC] Trying API TOC endpoint: %s\n", apiURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.GetHTTPClient().Do(req)
	if err != nil {
		fmt.Printf("[oreilly][FetchTOC] API TOC endpoint network error: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Printf("[oreilly][FetchTOC] API TOC status: %s\n", resp.Status)

	if resp.StatusCode == http.StatusOK {
		// Try to parse a TOC-style JSON payload
		var toc struct {
			Chapters []struct {
				Title string `json:"title"`
				Path  string `json:"path"`
			} `json:"chapters"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&toc); err != nil {
			fmt.Printf("[oreilly][FetchTOC] API TOC decode error: %v\n", err)
			return nil, fmt.Errorf("bad toc json: %v", err)
		}
		var chapters []Chapter
		for _, c := range toc.Chapters {
			chapters = append(chapters, Chapter{
				Title: c.Title,
				URL:   c.Path,
			})
		}
		fmt.Printf("[oreilly][FetchTOC] API TOC gave %d chapters.\n", len(chapters))
		return chapters, nil
	}

	// Fallback: parse HTML navigation.xhtml to extract chapter links
	tocURL := fmt.Sprintf("https://learning.oreilly.com/library/view/%s/%s/navigation.xhtml", slug, bookID)
	fmt.Printf("[oreilly][FetchTOC] Trying navigation.xhtml fallback: %s\n", tocURL)
	req, err = http.NewRequestWithContext(ctx, "GET", tocURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	resp, err = s.client.GetHTTPClient().Do(req)
	if err != nil {
		fmt.Printf("[oreilly][FetchTOC] navigation.xhtml network error: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Printf("[oreilly][FetchTOC] navigation.xhtml status: %s\n", resp.Status)

	// If navigation.xhtml is missing, try navigation.xhtml as fallback
	if resp.StatusCode != http.StatusOK {
		navURL := fmt.Sprintf("https://learning.oreilly.com/library/view/%s/%s/navigation.xhtml", slug, bookID)
		fmt.Printf("[oreilly][FetchTOC] Trying navigation.xhtml fallback: %s\n", navURL)
		req2, err2 := http.NewRequestWithContext(ctx, "GET", navURL, nil)
		if err2 != nil {
			return nil, err2
		}
		req2.Header.Set("Authorization", "Bearer "+jwt)
		resp2, err2 := s.client.GetHTTPClient().Do(req2)
		if err2 != nil {
			fmt.Printf("[oreilly][FetchTOC] navigation.xhtml network error: %v\n", err2)
			return nil, err2
		}
		defer resp2.Body.Close()
		fmt.Printf("[oreilly][FetchTOC] navigation.xhtml status: %s\n", resp2.Status)
		if resp2.StatusCode != http.StatusOK {
			fmt.Printf("[oreilly][FetchTOC] navigation.xhtml failed after navigation.xhtml\n")
			return nil, fmt.Errorf("failed to fetch TOC: %s then navigation.xhtml: %s", resp.Status, resp2.Status)
		}
		// Use the body/content of navigation.xhtml instead for parsing
		bodyBytes, err := io.ReadAll(resp2.Body)
		if err != nil {
			return nil, err
		}
		body := string(bodyBytes)
		// Find all links to XHTML chapters
		re := regexp.MustCompile(`<a[^>]+href="([^"]+\.xhtml)"[^>]*>(.*?)</a>`)
		matches := re.FindAllStringSubmatch(body, -1)
		var chapters []Chapter
		for _, match := range matches {
			rawURL := match[1]
			title := stripTags(match[2])
			if strings.HasSuffix(rawURL, ".xhtml") && !strings.Contains(rawURL, "index.xhtml") {
				chapters = append(chapters, Chapter{
					Title: htmlUnescape(title),
					URL:   rawURL,
				})
			}
		}
		fmt.Printf("[oreilly][FetchTOC] navigation.xhtml gave %d chapters\n", len(chapters))
		return chapters, nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	body := string(bodyBytes)

	// Find all links to XHTML chapters
	re := regexp.MustCompile(`<a[^>]+href="([^"]+\.xhtml)"[^>]*>(.*?)</a>`)
	matches := re.FindAllStringSubmatch(body, -1)

	var chapters []Chapter
	for _, match := range matches {
		rawURL := match[1]
		title := stripTags(match[2])
		// Only keep .xhtml links that likely are content, not toc, index, etc.
		if strings.HasSuffix(rawURL, ".xhtml") && !strings.Contains(rawURL, "index.xhtml") {
			chapters = append(chapters, Chapter{
				Title: htmlUnescape(title),
				URL:   rawURL,
			})
		}
	}
	fmt.Printf("[oreilly][FetchTOC] navigation.xhtml HTML gave %d chapters\n", len(chapters))
	return chapters, nil
}

// Helper to strip HTML tags (naive, good enough for simple TOCs).
func stripTags(s string) string {
	re := regexp.MustCompile("<[^>]*>")
	return re.ReplaceAllString(s, "")
}

// HTML entity unescape utility for barebones extraction.
func htmlUnescape(s string) string {
	replacer := strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'")
	return replacer.Replace(s)
}
