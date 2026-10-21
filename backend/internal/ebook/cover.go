package ebook

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	coverW = 1200
	coverH = 1600
)

var frenchMonths = []string{"janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"}
var frenchDays = []string{"dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"}

// FormatDateFR formats a date like "mercredi 21 octobre 2026".
func FormatDateFR(t time.Time) string {
	return fmt.Sprintf("%s %d %s %d", frenchDays[t.Weekday()], t.Day(), frenchMonths[t.Month()-1], t.Year())
}

func face(ttf []byte, size float64) (font.Face, error) {
	f, err := opentype.Parse(ttf)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
}

// GenerateCover renders a simple, high-contrast cover suited for e-ink screens.
func GenerateCover(title string, date time.Time, sources []string, articleCount int) ([]byte, error) {
	img := image.NewGray(image.Rect(0, 0, coverW, coverH))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	black := image.NewUniform(color.Black)
	white := image.NewUniform(color.White)
	gray := image.NewUniform(color.Gray{Y: 90})

	// Top band
	draw.Draw(img, image.Rect(0, 0, coverW, 520), black, image.Point{}, draw.Src)
	// Bottom rule
	draw.Draw(img, image.Rect(80, coverH-170, coverW-80, coverH-164), black, image.Point{}, draw.Src)

	titleFace, err := face(gobold.TTF, 96)
	if err != nil {
		return nil, err
	}
	smallBold, _ := face(gobold.TTF, 40)
	regular, _ := face(goregular.TTF, 40)
	big, _ := face(gobold.TTF, 150)

	d := &font.Drawer{Dst: img}

	// Kicker
	d.Src, d.Face = white, smallBold
	drawText(d, "RSS TO KINDLE", 80, 130)

	// Title (wrapped)
	d.Face = titleFace
	y := 260
	for _, line := range wrap(d, title, coverW-160, 3) {
		drawText(d, line, 80, y)
		y += 110
	}

	// Date
	d.Src, d.Face = black, smallBold
	drawText(d, strings.ToUpper(FormatDateFR(date)), 80, 640)

	// Article count
	d.Face = big
	drawText(d, fmt.Sprintf("%d", articleCount), 80, 840)
	d.Face = regular
	label := "articles"
	if articleCount == 1 {
		label = "article"
	}
	drawText(d, label, 80, 900)

	// Sources
	d.Src, d.Face = gray, regular
	y = 1010
	max := 7
	for i, s := range sources {
		if i == max {
			drawText(d, fmt.Sprintf("+ %d autres sources", len(sources)-max), 80, y)
			break
		}
		lines := wrap(d, "• "+s, coverW-160, 1)
		if len(lines) > 0 {
			drawText(d, lines[0], 80, y)
		}
		y += 56
	}

	// Footer
	d.Src, d.Face = black, regular
	drawText(d, date.Format("02/01/2006 · 15:04"), 80, coverH-90)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawText(d *font.Drawer, s string, x, y int) {
	d.Dot = fixed.P(x, y)
	d.DrawString(s)
}

// wrap breaks text into at most maxLines lines fitting width pixels.
func wrap(d *font.Drawer, text string, width, maxLines int) []string {
	words := strings.Fields(text)
	var lines []string
	cur := ""
	for _, w := range words {
		try := w
		if cur != "" {
			try = cur + " " + w
		}
		if d.MeasureString(try).Ceil() <= width || cur == "" {
			cur = try
			continue
		}
		lines = append(lines, cur)
		cur = w
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		last := lines[maxLines-1]
		for d.MeasureString(last+"…").Ceil() > width && len(last) > 1 {
			r := []rune(last)
			last = string(r[:len(r)-1])
		}
		lines[maxLines-1] = strings.TrimSpace(last) + "…"
	}
	// Hard-truncate single words that are too long.
	for i, l := range lines {
		for d.MeasureString(l).Ceil() > width && len([]rune(l)) > 1 {
			r := []rune(l)
			l = string(r[:len(r)-2]) + "…"
		}
		lines[i] = l
	}
	return lines
}
