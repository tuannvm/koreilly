package oreilly

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ProgressFn is an optional callback invoked periodically while the download
// is in progress.  The argument is the percentage completed in the range
// 0–100.  Pass nil if you do not care about progress events.
type ProgressFn func(percent float64)

// DownloadEPUB downloads the EPUB for a given book slug and saves it at
// destPath.  It authenticates using the supplied JWT (orm-jwt cookie value).
//
// Behaviour:
//   - Streams directly to a temporary file (same directory as destPath)
//     and renames on success (atomic write).
//   - Validates Content-Type starts with "application/epub".
//   - If server responds 404, returns os.ErrNotExist so caller can attempt
//     PDF fallback.
//
// The caller may supply a progress callback; if nil, no progress is reported.
func (s *Service) DownloadEPUB(ctx context.Context, jwt, slug, destPath string, progress ProgressFn) error {
	if jwt == "" {
		return fmt.Errorf("download: empty JWT token")
	}
	if slug == "" {
		return fmt.Errorf("download: empty book slug")
	}
	if destPath == "" {
		return fmt.Errorf("download: empty destination path")
	}

	endpoint := fmt.Sprintf("https://learning.oreilly.com/api/v2/epubs/%s.epub", slug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/epub+zip")

	resp, err := s.client.GetHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusNotFound:
		// Try PDF fallback if EPUB is not found
		pdfDest := strings.TrimSuffix(destPath, ".epub") + ".pdf"
		if pdfErr := s.DownloadPDF(ctx, jwt, slug, pdfDest, progress); pdfErr == nil {
			return nil
		} else {
			return fmt.Errorf("%w: epub and pdf not found (pdf fallback error: %v)", os.ErrNotExist, pdfErr)
		}
	default:
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	// Basic content-type sanity check
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/epub") {
		return fmt.Errorf("unexpected content-type %q", ct)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	// Create temp file alongside final destination for atomic rename
	tmpFile, err := os.CreateTemp(filepath.Dir(destPath), ".goreilly-*.epub")
	if err != nil {
		return err
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name()) // removed if rename below fails
	}()

	// Copy with progress
	var written int64
	contentLen := resp.ContentLength
	buf := make([]byte, 32*1024)

	lastEmit := time.Now()
	for {
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			nw, ew := tmpFile.Write(buf[0:nr])
			if ew != nil {
				return fmt.Errorf("write tmp: %w", ew)
			}
			if nw < nr {
				return fmt.Errorf("short write")
			}

			written += int64(nw)
			if progress != nil && contentLen > 0 {
				// throttle to 4 / second
				if time.Since(lastEmit) > 250*time.Millisecond || er == io.EOF {
					percent := float64(written) * 100 / float64(contentLen)
					progress(percent)
					lastEmit = time.Now()
				}
			}
		}
		if er != nil {
			if er == io.EOF {
				break
			}
			return fmt.Errorf("read body: %w", er)
		}
	}

	if err := tmpFile.Sync(); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	// Rename into place
	if err := os.Rename(tmpFile.Name(), destPath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

// DownloadPDF downloads the PDF for a given book slug and saves it at destPath.
// This is used as a fallback if EPUB is unavailable.
func (s *Service) DownloadPDF(ctx context.Context, jwt, slug, destPath string, progress ProgressFn) error {
	if jwt == "" {
		return fmt.Errorf("download: empty JWT token")
	}
	if slug == "" {
		return fmt.Errorf("download: empty book slug")
	}
	if destPath == "" {
		return fmt.Errorf("download: empty destination path")
	}
	endpoint := fmt.Sprintf("https://learning.oreilly.com/api/v2/pdfs/%s.pdf", slug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/pdf")

	resp, err := s.client.GetHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusNotFound:
		return fmt.Errorf("%w: pdf not found", os.ErrNotExist)
	default:
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/pdf") {
		return fmt.Errorf("unexpected PDF content-type %q", ct)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(destPath), ".goreilly-*.pdf")
	if err != nil {
		return err
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	var written int64
	contentLen := resp.ContentLength
	buf := make([]byte, 32*1024)
	lastEmit := time.Now()
	for {
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			nw, ew := tmpFile.Write(buf[0:nr])
			if ew != nil {
				return fmt.Errorf("write tmp: %w", ew)
			}
			if nw < nr {
				return fmt.Errorf("short write")
			}
			written += int64(nw)
			if progress != nil && contentLen > 0 {
				if time.Since(lastEmit) > 250*time.Millisecond || er == io.EOF {
					percent := float64(written) * 100 / float64(contentLen)
					progress(percent)
					lastEmit = time.Now()
				}
			}
		}
		if er != nil {
			if er == io.EOF {
				break
			}
			return fmt.Errorf("read body: %w", er)
		}
	}
	if err := tmpFile.Sync(); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpFile.Name(), destPath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
