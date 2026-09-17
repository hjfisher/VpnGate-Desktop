package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

// ICO format writer
type ICOHeader struct {
	Reserved  uint16
	Type      uint16
	Count     uint16
}

type ICODirEntry struct {
	Width       uint8
	Height      uint8
	ColorCount  uint8
	Reserved    uint8
	Planes      uint16
	BitCount    uint16
	SizeInBytes uint32
	Offset      uint32
}

func main() {
	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	var images [][]byte

	for _, size := range sizes {
		img := createIcon(size)
		var buf bytes.Buffer
		png.Encode(&buf, img)
		images = append(images, buf.Bytes())
	}

	// Write ICO file
	icoPath := "assets/icon.ico"
	f, err := os.Create(icoPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	header := ICOHeader{Reserved: 0, Type: 1, Count: uint16(len(sizes))}
	writeU16(f, header.Reserved)
	writeU16(f, header.Type)
	writeU16(f, header.Count)

	offset := uint32(6 + len(sizes)*16)
	for i, imgData := range images {
		entry := ICODirEntry{
			Width:       uint8(sizes[i]),
			Height:      uint8(sizes[i]),
			ColorCount:  0,
			Reserved:    0,
			Planes:      1,
			BitCount:    32,
			SizeInBytes: uint32(len(imgData)),
			Offset:      offset,
		}
		writeU8(f, entry.Width)
		writeU8(f, entry.Height)
		writeU8(f, entry.ColorCount)
		writeU8(f, entry.Reserved)
		writeU16(f, entry.Planes)
		writeU16(f, entry.BitCount)
		writeU32(f, entry.SizeInBytes)
		writeU32(f, entry.Offset)
		offset += uint32(len(imgData))
	}

	for _, imgData := range images {
		f.Write(imgData)
	}

	println("Created", icoPath)
}

func createIcon(size int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.Transparent}, image.Point{}, draw.Src)

	cx, cy := size/2, size/2
	r := size/2 - size/16

	// Background circle - blue gradient
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			dist := float64(x*x + y*y)
			if dist <= float64(r*r) {
				alpha := uint8(255 * (1 - dist/float64(r*r)))
				img.Set(cx+x, cy+y, color.NRGBA{R: 37, G: 99, B: 235, A: alpha})
			}
		}
	}

	// Shield - white
	shieldColor := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	points := []image.Point{
		{cx, size / 5}, {size - size/8, size*3/10}, {size - size/8, size*7/10},
		{cx, size - size/8}, {size/8, size*7/10}, {size/8, size*3/10},
	}
	drawPolygon(img, points, shieldColor)

	// Lock - blue
	lockColor := color.NRGBA{R: 37, G: 99, B: 235, A: 255}
	ls := size / 16
	lx, ly := cx-ls, cy+size/10
	for y := 0; y < 2*ls; y++ {
		for x := 0; x < 2*ls; x++ {
			if x >= ls/2 && x < 3*ls/2 {
				img.Set(lx+x, ly+y, lockColor)
			}
		}
	}
	for y := -ls; y < 0; y++ {
		for x := 0; x < 2*ls; x++ {
			if x == 0 || x == 2*ls-1 || y == -ls {
				img.Set(lx+x, ly+y, lockColor)
			}
		}
	}

	return img
}

func drawPolygon(img *image.RGBA, points []image.Point, c color.Color) {
	if len(points) < 3 {
		return
	}
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
		var xs []int
		for i := 0; i < len(points); i++ {
			p1 := points[i]
			p2 := points[(i+1)%len(points)]
			if (p1.Y <= y && p2.Y > y) || (p2.Y <= y && p1.Y > y) {
				x := p1.X + (y-p1.Y)*(p2.X-p1.X)/(p2.Y-p1.Y)
				xs = append(xs, x)
			}
		}
		for i := 0; i < len(xs); i += 2 {
			if i+1 < len(xs) {
				x1, x2 := xs[i], xs[i+1]
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

func writeU8(f *os.File, v uint8) {
	f.Write([]byte{byte(v)})
}

func writeU16(f *os.File, v uint16) {
	f.Write([]byte{byte(v), byte(v >> 8)})
}

func writeU32(f *os.File, v uint32) {
	f.Write([]byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)})
}