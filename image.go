// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2020 Mikhail Vladimirov
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package img1b is a fork of the standard Go image package modified for 1-bit images.
// Images are kept packed so they take up to 8 times less memory and may be processed
// faster.
//
// There is currently no img1b.Decode. To read a PNG file use Decode from img1b/png.
package img1b

import (
	"bytes"
	"image"
	"image/color"
	"math/bits"
)

// Image implements the image.PalettedImage interface and is mostly analogous to
// image.Paletted except that Pix is a bitmap, so only color indices 0 and 1 can be used.
type Image struct {
	// Pix is a bitmap of image pixels. Bytes represent up to 8 horizontally adjacent
	// pixels (there may be unused bits in the last byte of a row) with the most
	// significant bit corresponding to leftmost pixel.
	Pix []byte
	// Stride is the Pix stride (in bytes) between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect image.Rectangle
	// Palette is the image's palette.
	Palette color.Palette
}

// At returns the color of the pixel at (x, y).
func (p *Image) At(x, y int) color.Color {
	if len(p.Palette) == 0 {
		return nil
	}
	if !(image.Point{x, y}.In(p.Rect)) {
		return p.Palette[0]
	}
	i, b := p.PixBitOffset(x, y)
	return p.Palette[(p.Pix[i]>>b)&1]
}

// PixBitOffset returns the index of the byte of Pix that corresponds to
// the pixel at (x, y) and bit offset (7 for MSB) in that byte.
func (p *Image) PixBitOffset(x, y int) (ofs, bit int) {
	ofs = (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)/8
	bit = 7 - (x-p.Rect.Min.X)%8
	return
}

// Bounds returns the domain for which At can return non-zero color.
// The bounds do not necessarily contain the point (0, 0).
func (p *Image) Bounds() image.Rectangle { return p.Rect }

// ColorModel returns the Image's color model.
func (p *Image) ColorModel() color.Model { return p.Palette }

// ColorIndexAt returns the palette index of the pixel at (x, y).
func (p *Image) ColorIndexAt(x, y int) uint8 {
	if !(image.Point{x, y}.In(p.Rect)) {
		return 0
	}
	i, b := p.PixBitOffset(x, y)
	return (p.Pix[i] >> b) & 1
}

// SetColorIndex sets color index for the pixel at (x, y). Index should be 0 or 1.
func (p *Image) SetColorIndex(x, y int, index uint8) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}
	i, b := p.PixBitOffset(x, y)
	if index == 0 {
		p.Pix[i] &^= 1 << b
	} else {
		p.Pix[i] |= 1 << b
	}
}

// mul2NonNeg returns (x * y), unless at least one argument is negative or
// if the computation overflows the int type, in which case it returns -1.
func mul2NonNeg(x int, y int) int {
	if (x < 0) || (y < 0) {
		return -1
	}
	hi, lo := bits.Mul64(uint64(x), uint64(y))
	if hi != 0 {
		return -1
	}
	a := int(lo)
	if (a < 0) || (uint64(a) != lo) {
		return -1
	}
	return a
}

// New returns a new Image with given dimensions and palette.
func New(r image.Rectangle, p color.Palette) *Image {
	w, h := r.Dx(), r.Dy()
	stride := (w + 7) / 8
	bytes := mul2NonNeg(h, stride)
	if w < 0 || bytes < 0 {
		panic("img1b.New: Rectangle has huge or negative dimensions")
	}
	pix := make([]byte, bytes)
	return &Image{pix, stride, r, p}
}

// SubImage returns an image representing the portion of the image p visible
// through r. The returned value shares pixels with the original image. Left edge
// has to be byte aligned.
func (p *Image) SubImage(r image.Rectangle) *Image {
	r = r.Intersect(p.Rect)
	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Pix[i:] expression below can panic.
	if r.Empty() {
		return &Image{
			Palette: p.Palette,
		}
	}
	i, b := p.PixBitOffset(r.Min.X, r.Min.Y)
	if b != 7 {
		panic("img1b.SubImage: left edge is not byte aligned")
	}
	return &Image{
		Pix:     p.Pix[i:],
		Stride:  p.Stride,
		Rect:    r,
		Palette: p.Palette,
	}
}

// Opaque scans the entire image and reports whether it is fully opaque.
func (p *Image) Opaque() bool {
	ts := 0 // transparent indices+1 sum
	for i, c := range p.Palette {
		if i > 1 {
			// skip inaccessible colors
			break
		}
		_, _, _, a := c.RGBA()
		if a != 0xffff {
			ts += i + 1
		}
	}

	var ob byte // opaque byte
	switch ts {
	case 0:
		return true // no transparent colors
	case 1:
		ob = 0xff
	case 2:
		ob = 0
	case 3:
		return false // both colors are transparent
	}

	i0, i1 := 0, p.Rect.Dx()/8 // whole byte indices
	bc := i1 - i0
	tm := byte(0xff) << (8 - p.Rect.Dx()%8) // tail mask
	for y := p.Rect.Min.Y; y < p.Rect.Max.Y; y++ {
		if bytes.Count(p.Pix[i0:i1], []byte{ob}) != bc {
			return false
		}
		if tm != 0 && p.Pix[i1]&tm != ob&tm {
			return false
		}
		i0 += p.Stride
		i1 += p.Stride
	}
	return true
}
