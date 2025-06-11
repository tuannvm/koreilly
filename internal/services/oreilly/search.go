package oreilly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// SearchResult represents a partial structure returned by the O’Reilly
// /api/v2/search/ endpoint. We only keep the fields we currently need.
type SearchResult struct {
	Count   int `json:"count"`
	Results []struct {
		Title  string `json:"title"`
		Slug   string `json:"slug"`
		Author string `json:"author"`
	} `json:"results"`
	Next string `json:"next"`
}

// SearchBooks queries the O’Reilly public search API.
//
// Example endpoint (undocumented but stable for years):
//
//	https://learning.oreilly.com/api/v2/search/?query=kubernetes&field=title&limit=5
//
// The user must provide a valid JWT cookie (orm-jwt) which we pass
// as an Authorization header.
//
// The function returns a SearchResult or an error if the request fails.
func (s *Service) SearchBooks(ctx context.Context, jwt, query string, limit int) (*SearchResult, error) {
	if limit <= 0 {
		limit = 5
	}
	// Build the URL manually to keep it simple and avoid extra structs.
	endpoint := fmt.Sprintf("https://learning.oreilly.com/api/v2/search/?query=%s&field=title&limit=%d",
		url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.GetHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed: HTTP %d", resp.StatusCode)
	}

	var sr SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	return &sr, nil
}
