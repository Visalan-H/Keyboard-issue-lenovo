package main

import (
	"encoding/binary"
	"testing"
)

func TestIconImage(t *testing.T) {
	for _, size := range []int{16, 20, 24, 32} {
		img := iconImage(size, false)
		mask := (size + 31) / 32 * 4 * size
		if want := 40 + size*size*4 + mask; len(img) != want {
			t.Fatalf("size %d: %d bytes, want %d", size, len(img), want)
		}
		if w := binary.LittleEndian.Uint32(img[4:]); int(w) != size {
			t.Errorf("width %d", w)
		}
		if h := binary.LittleEndian.Uint32(img[8:]); int(h) != size*2 {
			t.Errorf("height %d, want twice the size", h)
		}
		alphaAt := func(x, y int) uint8 { return img[40+((size-1-y)*size+x)*4+3] }
		if alphaAt(0, 0) > 64 {
			t.Errorf("size %d: corner should be (mostly) transparent, alpha %d", size, alphaAt(0, 0))
		}
		if alphaAt(size/2, size/2) != 255 {
			t.Errorf("size %d: centre should be opaque", size)
		}
	}
	if string(iconImage(16, false)) == string(iconImage(16, true)) {
		t.Error("paused icon should look different")
	}
}
