package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"fyne.io/fyne/v2"
)

// AppIcon returns a simple VPN-themed icon as a fyne resource.
func AppIcon() fyne.Resource {
	// Create a 64x64 icon
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.Transparent}, image.Point{}, draw.Src)

	// Background circle - blue gradient
	centerX, centerY := 32, 32
	radius := 28
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			dist := float64(x*x + y*y)
			if dist <= float64(radius*radius) {
				// Blue gradient
				alpha := uint8(255 * (1 - dist/float64(radius*radius)))
				img.Set(centerX+x, centerY+y, color.NRGBA{
					R: 37, G: 99, B: 235, A: alpha,
				})
			}
		}
	}

	// Shield shape - white
	shieldColor := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	// Simple shield polygon
	points := []image.Point{
		{32, 14}, {50, 20}, {50, 44}, {32, 56}, {14, 44}, {14, 20},
	}
	drawPolygon(img, points, shieldColor)

	// Lock icon inside shield
	lockColor := color.NRGBA{R: 37, G: 99, B: 235, A: 255}
	// Lock body
	for y := 30; y <= 42; y++ {
		for x := 26; x <= 38; x++ {
			if x >= 28 && x <= 36 {
				img.Set(x, y, lockColor)
			}
		}
	}
	// Lock shackle
	for y := 26; y <= 30; y++ {
		for x := 29; x <= 35; x++ {
			if (x == 29 || x == 35) || y == 26 {
				img.Set(x, y, lockColor)
			}
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return fyne.NewStaticResource("app-icon.png", buf.Bytes())
}

func drawPolygon(img *image.RGBA, points []image.Point, c color.Color) {
	if len(points) < 3 {
		return
	}
	// Simple fill using scanline
	minY, maxY := points[0].Y, points[0].Y
	for _, p := range points {
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	for y := minY; y <= maxY; y++ {
		var intersections []int
		for i := 0; i < len(points); i++ {
			p1 := points[i]
			p2 := points[(i+1)%len(points)]
			if (p1.Y <= y && p2.Y > y) || (p2.Y <= y && p1.Y > y) {
				x := p1.X + (y-p1.Y)*(p2.X-p1.X)/(p2.Y-p1.Y)
				intersections = append(intersections, x)
			}
		}
		for i := 0; i < len(intersections); i += 2 {
			if i+1 < len(intersections) {
				x1, x2 := intersections[i], intersections[i+1]
				if x1 > x2 {
					x1, x2 = x2, x1
				}
				for x := x1; x <= x2; x++ {
					img.Set(x, y, c)
				}
			}
		}
	}
}