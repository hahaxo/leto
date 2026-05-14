// Package filex provides file and asset helpers.
package filex

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

const (
	defaultUploadPath      = "uploads"
	defaultPublicURLPrefix = "/uploads"
	defaultHTTPTimeout     = 15 * time.Second
)

// ImageService downloads remote images and builds Markdown image references.
type ImageService struct {
	uploadPath      string
	publicURLPrefix string
	httpClient      *http.Client
}

type imageServiceConfig struct {
	publicURLPrefix string
	httpClient      *http.Client
	timeout         time.Duration
}

// ImageServiceOption configures ImageService.
type ImageServiceOption func(*imageServiceConfig)

// WithPublicURLPrefix configures the public URL prefix used for downloaded images.
func WithPublicURLPrefix(prefix string) ImageServiceOption {
	return func(config *imageServiceConfig) {
		if strings.TrimSpace(prefix) != "" {
			config.publicURLPrefix = strings.TrimRight(strings.TrimSpace(prefix), "/")
		}
	}
}

// WithHTTPClient configures the HTTP client used to download images.
func WithHTTPClient(client *http.Client) ImageServiceOption {
	return func(config *imageServiceConfig) {
		if client != nil {
			config.httpClient = client
		}
	}
}

// WithTimeout configures the default HTTP client timeout.
func WithTimeout(timeout time.Duration) ImageServiceOption {
	return func(config *imageServiceConfig) {
		if timeout > 0 {
			config.timeout = timeout
		}
	}
}

// NewImageService creates an ImageService.
func NewImageService(uploadPath string, options ...ImageServiceOption) *ImageService {
	uploadPath = strings.TrimSpace(uploadPath)
	if uploadPath == "" {
		uploadPath = defaultUploadPath
	}

	config := imageServiceConfig{
		publicURLPrefix: defaultPublicURLPrefix,
		timeout:         defaultHTTPTimeout,
	}
	for _, option := range options {
		option(&config)
	}

	client := config.httpClient
	if client == nil {
		client = &http.Client{Timeout: config.timeout}
	}

	return &ImageService{
		uploadPath:      uploadPath,
		publicURLPrefix: config.publicURLPrefix,
		httpClient:      client,
	}
}

// MarkdownImagesForURLs converts newline-separated image URLs into Markdown image references.
func (s *ImageService) MarkdownImagesForURLs(ctx context.Context, imageURLs string) []string {
	lines := strings.Split(imageURLs, "\n")
	markdownImages := make([]string, 0, len(lines))
	for _, line := range lines {
		rawURL := strings.TrimSpace(line)
		if rawURL == "" {
			continue
		}
		imageURL, err := s.DownloadImageURL(ctx, rawURL)
		if err != nil {
			imageURL = rawURL
		}
		markdownImages = append(markdownImages, fmt.Sprintf("![image](%s)", imageURL))
	}
	return markdownImages
}

// DownloadImageURL downloads an image URL into the upload directory and returns its public URL.
func (s *ImageService) DownloadImageURL(ctx context.Context, rawURL string) (string, error) {
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return "", err
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("unsupported image url scheme: %s", parsedURL.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("download image url %q: status %d", rawURL, resp.StatusCode)
	}

	now := time.Now()
	year := now.Format("2006")
	filename := downloadedImageFilename(parsedURL, resp.Header.Get("Content-Type"), now)
	dir := filepath.Join(s.uploadPath, "img", year)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	filePath := filepath.Join(dir, filename)
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", err
	}

	return joinPublicURL(s.publicURLPrefix, "img", year, filename), nil
}

func downloadedImageFilename(parsedURL *url.URL, contentType string, now time.Time) string {
	base := path.Base(parsedURL.Path)
	if base == "." || base == "/" {
		base = "image"
	}

	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		name = "image"
	}
	if ext == "" {
		ext = extensionFromContentType(contentType)
	}
	if ext == "" {
		ext = ".jpg"
	}

	return fmt.Sprintf("%d_%s%s", now.UnixNano(), safeFilenamePart(name), safeExtension(ext))
}

func extensionFromContentType(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	extensions, err := mime.ExtensionsByType(mediaType)
	if err != nil || len(extensions) == 0 {
		return ""
	}
	return extensions[0]
}

func safeFilenamePart(value string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteByte('_')
	}
	name := strings.Trim(builder.String(), "._-")
	if name == "" {
		return "image"
	}
	return name
}

func safeExtension(ext string) string {
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	ext = safeFilenamePart(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if ext == "." {
		return ".jpg"
	}
	return ext
}

func joinPublicURL(prefix string, elements ...string) string {
	suffix := path.Join(elements...)
	if prefix == "" {
		return "/" + suffix
	}
	return strings.TrimRight(prefix, "/") + "/" + suffix
}
