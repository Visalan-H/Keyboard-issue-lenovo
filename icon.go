package main

// The tray icon is drawn in code rather than shipped as a file: a small
// keycap, blue when running and grey when paused. The bytes are in the
// format CreateIconFromResourceEx expects (a BITMAPINFOHEADER, 32-bit
// BGRA pixels bottom-up, then a 1-bit AND mask).

import (
	"encoding/binary"
	"math"
)

type rgb struct{ r, g, b uint8 }

var (
	iconRunning = [2]rgb{{0x1e, 0x3a, 0x8a}, {0x60, 0xa5, 0xfa}} // edge, face
	iconPaused  = [2]rgb{{0x4b, 0x55, 0x63}, {0x9c, 0xa3, 0xaf}}
)

// iconImage returns a size x size icon.
func iconImage(size int, paused bool) []byte {
	colors := iconRunning
	if paused {
		colors = iconPaused
	}
	s := float64(size)
	// The whole cap is the darker edge; the lighter top face sits inside
	// it, higher up, so it reads as a key seen from the front.
	edge := roundRect{x0: 0.5, y0: 0.5, x1: s - 0.5, y1: s - 0.5, r: s * 0.22}
	face := roundRect{x0: s * 0.16, y0: s * 0.1, x1: s * 0.84, y1: s * 0.7, r: s * 0.14}

	const headerSize = 40
	maskStride := (size + 31) / 32 * 4
	buf := make([]byte, headerSize+size*size*4+maskStride*size)

	// BITMAPINFOHEADER. The height counts the colour image and the mask.
	le := binary.LittleEndian
	le.PutUint32(buf[0:], headerSize)
	le.PutUint32(buf[4:], uint32(size))
	le.PutUint32(buf[8:], uint32(size*2))
	le.PutUint16(buf[12:], 1)  // planes
	le.PutUint16(buf[14:], 32) // bits per pixel
	le.PutUint32(buf[20:], uint32(size*size*4))

	px := buf[headerSize:]
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			a := edge.coverage(fx, fy)
			c := colors[0]
			if f := face.coverage(fx, fy); f > 0 {
				c = mix(colors[0], colors[1], f)
			}
			// Rows are stored bottom-up.
			i := ((size-1-y)*size + x) * 4
			px[i+0] = c.b
			px[i+1] = c.g
			px[i+2] = c.r
			px[i+3] = uint8(math.Round(a * 255))
		}
	}
	// The AND mask stays all zero: the alpha channel does the work.
	return buf
}

type roundRect struct{ x0, y0, x1, y1, r float64 }

// coverage is how much of the pixel at (x, y) is inside the shape, 0-1,
// which gives smooth anti-aliased edges.
func (q roundRect) coverage(x, y float64) float64 {
	cx, cy := (q.x0+q.x1)/2, (q.y0+q.y1)/2
	hx, hy := (q.x1-q.x0)/2-q.r, (q.y1-q.y0)/2-q.r
	dx := math.Max(math.Abs(x-cx)-hx, 0)
	dy := math.Max(math.Abs(y-cy)-hy, 0)
	d := math.Hypot(dx, dy) - q.r // < 0 inside
	return math.Min(math.Max(0.5-d, 0), 1)
}

func mix(a, b rgb, t float64) rgb {
	l := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return rgb{l(a.r, b.r), l(a.g, b.g), l(a.b, b.b)}
}
