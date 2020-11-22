// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2020 Mikhail Vladimirov
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package png

import (
	"bytes"
	"fmt"
	"github.com/mi-v/img1b"
	"image"
	"image/color"
	gopng "image/png"
	"io/ioutil"
	"testing"
)

func diff(m0, m1 *img1b.Image) error {
	b0, b1 := m0.Bounds(), m1.Bounds()
	if !b0.Size().Eq(b1.Size()) {
		return fmt.Errorf("dimensions differ: %v vs %v", b0, b1)
	}
	dx := b1.Min.X - b0.Min.X
	dy := b1.Min.Y - b0.Min.Y
	for y := b0.Min.Y; y < b0.Max.Y; y++ {
		for x := b0.Min.X; x < b0.Max.X; x++ {
			c0 := m0.At(x, y)
			c1 := m1.At(x+dx, y+dy)
			r0, g0, b0, a0 := c0.RGBA()
			r1, g1, b1, a1 := c1.RGBA()
			if r0 != r1 || g0 != g1 || b0 != b1 || a0 != a1 {
				return fmt.Errorf("colors differ at (%d, %d): %v vs %v", x, y, c0, c1)
			}
		}
	}
	return nil
}

func encodeDecode(m *img1b.Image) (*img1b.Image, error) {
	var b bytes.Buffer
	err := Encode(&b, m)
	if err != nil {
		return nil, err
	}
	return Decode(&b)
}

func TestWriter(t *testing.T) {
	// The filenames variable is declared in reader_test.go.
	names := filenames
	for _, fn := range names {
		qfn := "testdata/pngsuite/" + fn + ".png"
		// Read the image.
		m0, err := readPNG(qfn)
		if err != nil {
			t.Error(fn, err)
			continue
		}
		// Read the image again, encode it, and decode it.
		m1, err := readPNG(qfn)
		if err != nil {
			t.Error(fn, err)
			continue
		}
		m2, err := encodeDecode(m1)
		if err != nil {
			t.Error(fn, err)
			continue
		}
		// Compare the two.
		err = diff(m0, m2)
		if err != nil {
			t.Error(fn, err)
			continue
		}
	}
}

func TestWriterLevels(t *testing.T) {
	p := color.Palette{color.Black, color.White}
	m := img1b.New(image.Rect(0, 0, 100, 100), p)

	var b1, b2 bytes.Buffer
	if err := (&Encoder{}).Encode(&b1, m); err != nil {
		t.Fatal(err)
	}
	noenc := &Encoder{CompressionLevel: NoCompression}
	if err := noenc.Encode(&b2, m); err != nil {
		t.Fatal(err)
	}

	if b2.Len() <= b1.Len() {
		t.Error("DefaultCompression encoding was larger than NoCompression encoding")
	}
	if _, err := Decode(&b1); err != nil {
		t.Error("cannot decode DefaultCompression")
	}
	if _, err := Decode(&b2); err != nil {
		t.Error("cannot decode NoCompression")
	}
}

func TestSubImage(t *testing.T) {
	p := color.Palette{color.Black, color.White}
	m0 := img1b.New(image.Rect(0, 0, 256, 256), p)
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			m0.SetColorIndex(x, y, (uint8(x*y)&128)>>7)
		}
	}
	m0 = m0.SubImage(image.Rect(48, 30, 250, 130))
	m1, err := encodeDecode(m0)
	if err != nil {
		t.Error(err)
		return
	}
	err = diff(m0, m1)
	if err != nil {
		t.Error(err)
		return
	}
}

type pool struct {
	b *EncoderBuffer
}

func (p *pool) Get() *EncoderBuffer {
	return p.b
}

func (p *pool) Put(b *EncoderBuffer) {
	p.b = b
}

func BenchmarkEncode(b *testing.B) {
	img := img1b.New(image.Rect(0, 0, 640, 480), color.Palette{
		color.Black,
		color.White,
	})
	b.SetBytes(640 * 480 / 8)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Encode(ioutil.Discard, img)
	}
}

func BenchmarkEncodeWithBufferPool(b *testing.B) {
	img := img1b.New(image.Rect(0, 0, 640, 480), color.Palette{
		color.Black,
		color.White,
	})
	e := Encoder{
		BufferPool: &pool{},
	}
	b.SetBytes(640 * 480 / 8)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Encode(ioutil.Discard, img)
	}
}

func BenchmarkEncodeStock(b *testing.B) {
	img := image.NewPaletted(image.Rect(0, 0, 640, 480), color.Palette{
		color.Black,
		color.White,
	})
	b.SetBytes(640 * 480 / 8)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gopng.Encode(ioutil.Discard, img)
	}
}
