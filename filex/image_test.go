package filex

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDownloadImageURLWritesFileAndReturnsPublicURL(t *testing.T) {
	t.Parallel()

	const body = "image-bytes"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	uploadPath := t.TempDir()
	service := NewImageService(uploadPath, WithPublicURLPrefix("/static"))

	publicURL, err := service.DownloadImageURL(context.Background(), server.URL+"/cat")
	if err != nil {
		t.Fatalf("DownloadImageURL() error = %v", err)
	}

	year := time.Now().Format("2006")
	if !strings.HasPrefix(publicURL, "/static/img/"+year+"/") {
		t.Fatalf("DownloadImageURL() publicURL = %q, want /static/img/%s prefix", publicURL, year)
	}
	if !strings.HasSuffix(publicURL, "_cat.png") {
		t.Fatalf("DownloadImageURL() publicURL = %q, want _cat.png suffix", publicURL)
	}

	matches, err := filepath.Glob(filepath.Join(uploadPath, "img", year, "*_cat.png"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("downloaded files = %v, want exactly one *_cat.png", matches)
	}

	gotBody, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(gotBody) != body {
		t.Fatalf("downloaded file body = %q, want %q", gotBody, body)
	}
}

func TestDownloadImageURLRejectsInvalidInputsAndStatuses(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "failed", http.StatusInternalServerError)
	}))
	defer server.Close()

	service := NewImageService(t.TempDir())

	tests := []struct {
		name   string
		rawURL string
	}{
		{name: "invalid url", rawURL: "://bad-url"},
		{name: "unsupported scheme", rawURL: "file:///tmp/image.jpg"},
		{name: "non 2xx status", rawURL: server.URL + "/image.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := service.DownloadImageURL(context.Background(), tt.rawURL); err == nil {
				t.Fatalf("DownloadImageURL(%q) error = nil, want error", tt.rawURL)
			}
		})
	}
}

func TestMarkdownImagesForURLsFallsBackToRawURLOnDownloadError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok" {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("ok"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	service := NewImageService(t.TempDir(), WithPublicURLPrefix("/media"))
	rawMissingURL := server.URL + "/missing"
	images := service.MarkdownImagesForURLs(context.Background(), "\n"+server.URL+"/ok\n \n"+rawMissingURL+"\n")

	if len(images) != 2 {
		t.Fatalf("MarkdownImagesForURLs() len = %d, want 2: %v", len(images), images)
	}
	if !strings.HasPrefix(images[0], "![image](/media/img/") || !strings.HasSuffix(images[0], "_ok.png)") {
		t.Fatalf("MarkdownImagesForURLs()[0] = %q, want downloaded media URL", images[0])
	}
	if images[1] != "![image]("+rawMissingURL+")" {
		t.Fatalf("MarkdownImagesForURLs()[1] = %q, want raw URL fallback", images[1])
	}
}

func TestFilenameHelpers(t *testing.T) {
	t.Parallel()

	now := time.Unix(123, 456)

	t.Run("content type extension", func(t *testing.T) {
		filename := downloadedImageFilename(mustParseURL(t, "https://example.com/path/photo"), "image/png", now)
		if filename != "123000000456_photo.png" {
			t.Fatalf("downloadedImageFilename() = %q, want %q", filename, "123000000456_photo.png")
		}
	})

	t.Run("safe filename", func(t *testing.T) {
		filename := downloadedImageFilename(mustParseURL(t, "https://example.com/path/a%20b%20c.JPG"), "", now)
		if filename != "123000000456_a_b_c.jpg" {
			t.Fatalf("downloadedImageFilename() = %q, want %q", filename, "123000000456_a_b_c.jpg")
		}
	})
}

func TestWithHTTPClientTakesPrecedenceOverTimeout(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("blocked")
	service := NewImageService(
		t.TempDir(),
		WithTimeout(time.Nanosecond),
		WithHTTPClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, wantErr
		}).client()),
	)

	_, err := service.DownloadImageURL(context.Background(), "https://example.com/image.jpg")
	if !errors.Is(err, wantErr) {
		t.Fatalf("DownloadImageURL() error = %v, want %v", err, wantErr)
	}
}

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		t.Fatalf("ParseRequestURI() error = %v", err)
	}
	return parsedURL
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func (fn roundTripFunc) client() *http.Client {
	return &http.Client{Transport: fn}
}
