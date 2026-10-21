// Package imageproc downloads and converts images into Kindle-friendly JPEGs.
package imageproc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"

	// Register decoders.
	_ "image/gif"
	_ "image/png"

	"github.com/benoit/rss2kindle/internal/extract"
	"github.com/disintegration/imaging"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

// ErrTooSmall marks images that are likely icons / tracking pixels.
var ErrTooSmall = errors.New("image too small")

// Options controls conversion.
type Options struct {
	MaxWidth  int
	MaxHeight int
	Grayscale bool
	Quality   int
}

// Download fetches an image (max 15 MB).
func Download(ctx context.Context, client *http.Client, url, referer string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", extract.UserAgent)
	req.Header.Set("Accept", "image/jpeg,image/png,image/gif,image/webp,image/*;q=0.8")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 15<<20))
}

// Convert decodes any supported image and re-encodes it as an optimised JPEG.
func Convert(data []byte, o Options) ([]byte, int, int, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("decode: %w", err)
	}
	b := img.Bounds()
	if b.Dx() < 48 || b.Dy() < 48 {
		return nil, 0, 0, ErrTooSmall
	}
	if o.MaxWidth <= 0 {
		o.MaxWidth = 600
	}
	if o.MaxHeight <= 0 {
		o.MaxHeight = 800
	}
	if o.Quality <= 0 {
		o.Quality = 75
	}
	if b.Dx() > o.MaxWidth || b.Dy() > o.MaxHeight {
		img = imaging.Fit(img, o.MaxWidth, o.MaxHeight, imaging.Lanczos)
	}
	// Flatten transparency onto white (JPEG has no alpha).
	bg := imaging.New(img.Bounds().Dx(), img.Bounds().Dy(), image.White.C)
	img = imaging.Overlay(bg, img, image.Pt(0, 0), 1.0)
	if o.Grayscale {
		img = imaging.Grayscale(img)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: o.Quality}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), img.Bounds().Dx(), img.Bounds().Dy(), nil
}
