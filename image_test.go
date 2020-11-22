// Copyright 2011 The Go Authors. All rights reserved.
// Copyright 2020 Mikhail Vladimirov
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package img1b

import (
	"image"
	"image/color"
	"testing"
)

func cmp(cm color.Model, c0, c1 color.Color) bool {
	r0, g0, b0, a0 := cm.Convert(c0).RGBA()
	r1, g1, b1, a1 := cm.Convert(c1).RGBA()
	return r0 == r1 && g0 == g1 && b0 == b1 && a0 == a1
}

func TestImage(t *testing.T) {
	m := New(image.Rect(0, 0, 14, 14), color.Palette{
		color.Transparent,
		color.Opaque,
	})

	// Check for right initial conditions.
	if !image.Rect(0, 0, 14, 14).Eq(m.Bounds()) {
		t.Errorf("%T: want bounds %v, got %v", m, image.Rect(0, 0, 14, 14), m.Bounds())
		return
	}
	if !cmp(m.ColorModel(), color.Transparent, m.At(7, 3)) {
		t.Errorf("%T: at (7, 3), want a zero color, got %v", m, m.At(7, 3))
		return
	}
	if !cmp(m.ColorModel(), color.Transparent, m.At(8, 3)) {
		t.Errorf("%T: at (8, 3), want a zero color, got %v", m, m.At(8, 3))
		return
	}
	if !cmp(m.ColorModel(), color.Transparent, m.At(9, 3)) {
		t.Errorf("%T: at (9, 3), want a zero color, got %v", m, m.At(9, 3))
		return
	}
	if !cmp(m.ColorModel(), color.Transparent, m.At(10, 3)) {
		t.Errorf("%T: at (10, 3), want a zero color, got %v", m, m.At(10, 3))
		return
	}

	// Check that SetColorIndex works both ways and does not spill into adjacent pixels.
	m.SetColorIndex(8, 3, 1)
	if !cmp(m.ColorModel(), color.Transparent, m.At(7, 3)) {
		t.Errorf("%T: at (7, 3), want a zero color, got %v", m, m.At(7, 3))
		return
	}
	if !cmp(m.ColorModel(), color.Opaque, m.At(8, 3)) {
		t.Errorf("%T: at (8, 3), want a non-zero color, got %v", m, m.At(8, 3))
		return
	}
	if !cmp(m.ColorModel(), color.Transparent, m.At(9, 3)) {
		t.Errorf("%T: at (9, 3), want a zero color, got %v", m, m.At(9, 3))
		return
	}

	m.SetColorIndex(9, 3, 1)
	if !cmp(m.ColorModel(), color.Opaque, m.At(8, 3)) {
		t.Errorf("%T: at (8, 3), want a non-zero color, got %v", m, m.At(8, 3))
		return
	}
	if !cmp(m.ColorModel(), color.Opaque, m.At(9, 3)) {
		t.Errorf("%T: at (9, 3), want a non-zero color, got %v", m, m.At(9, 3))
		return
	}
	if !cmp(m.ColorModel(), color.Transparent, m.At(10, 3)) {
		t.Errorf("%T: at (10, 3), want a zero color, got %v", m, m.At(10, 3))
		return
	}

	m.SetColorIndex(9, 3, 0)
	if !cmp(m.ColorModel(), color.Opaque, m.At(8, 3)) {
		t.Errorf("%T: at (8, 3), want a non-zero color, got %v", m, m.At(8, 3))
		return
	}
	if !cmp(m.ColorModel(), color.Transparent, m.At(9, 3)) {
		t.Errorf("%T: at (9, 3), want a zero color, got %v", m, m.At(9, 3))
		return
	}
	if !cmp(m.ColorModel(), color.Transparent, m.At(10, 3)) {
		t.Errorf("%T: at (10, 3), want a zero color, got %v", m, m.At(10, 3))
		return
	}

	// SubImage tests.
	m.SetColorIndex(8, 3, 1)
	if !m.SubImage(image.Rect(8, 3, 9, 4)).Opaque() {
		t.Errorf("%T: at (8, 3) was not opaque", m)
		return
	}

	m = m.SubImage(image.Rect(8, 2, 13, 8))
	if !image.Rect(8, 2, 13, 8).Eq(m.Bounds()) {
		t.Errorf("%T: sub-image want bounds %v, got %v", m, image.Rect(8, 2, 13, 8), m.Bounds())
		return
	}
	if !cmp(m.ColorModel(), color.Opaque, m.At(8, 3)) {
		t.Errorf("%T: sub-image at (8, 3), want a non-zero color, got %v", m, m.At(8, 3))
		return
	}
	if !cmp(m.ColorModel(), color.Transparent, m.At(8, 4)) {
		t.Errorf("%T: sub-image at (8, 4), want a zero color, got %v", m, m.At(8, 4))
		return
	}
	m.SetColorIndex(8, 4, 1)
	if !cmp(m.ColorModel(), color.Opaque, m.At(8, 4)) {
		t.Errorf("%T: sub-image at (8, 4), want a non-zero color, got %v", m, m.At(8, 4))
		return
	}
	// Test that taking an empty sub-image starting at a corner does not panic.
	m.SubImage(image.Rect(0, 0, 0, 0))
	m.SubImage(image.Rect(14, 0, 14, 0))
	m.SubImage(image.Rect(0, 10, 0, 14))
	m.SubImage(image.Rect(14, 10, 14, 14))
}

func TestNewBadRectangle(t *testing.T) {
	// call calls f(r) and reports whether it ran without panicking.
	call := func(f func(image.Rectangle), r image.Rectangle) (ok bool) {
		defer func() {
			if recover() != nil {
				ok = false
			}
		}()
		f(r)
		return true
	}

	// Calling New(r) should fail (panic, since New doesn't return an error)
	// unless r's width and height are both non-negative.
	f := func(r image.Rectangle) { New(r, color.Palette{color.Black, color.White}) }
	for _, negDx := range []bool{false, true} {
		for _, negDy := range []bool{false, true} {
			r := image.Rectangle{
				Min: image.Point{15, 28},
				Max: image.Point{16, 29},
			}
			if negDx {
				r.Max.X = 14
			}
			if negDy {
				r.Max.Y = 27
			}

			got := call(f, r)
			want := !negDx && !negDy
			if got != want {
				t.Errorf("New: negDx=%t, negDy=%t: got %t, want %t",
					negDx, negDy, got, want)
			}
		}
	}

	// Passing a Rectangle whose width and height is MaxInt should also fail
	// (panic), due to overflow.
	{
		zeroAsUint := uint(0)
		maxUint := zeroAsUint - 1
		maxInt := int(maxUint / 2)
		got := call(f, image.Rectangle{
			Min: image.Point{0, 0},
			Max: image.Point{maxInt, maxInt},
		})
		if got {
			t.Error("New: overflow: got ok, want !ok")
		}
	}
}

func TestSubimageBadRectangle(t *testing.T) {
	// call calls f(r) and reports whether it ran without panicking.
	call := func(f func(image.Rectangle), r image.Rectangle) (ok bool) {
		defer func() {
			if recover() != nil {
				ok = false
			}
		}()
		f(r)
		return true
	}

	m := New(image.Rect(0, 0, 14, 14), color.Palette{
		color.Transparent,
		color.Opaque,
	})

	// Calling SubImage(r) should fail (panic, since SubImage doesn't return
	// an error) unless r.Min.X is byte aligned.
	f := func(r image.Rectangle) { m.SubImage(r) }

	{
		r := image.Rectangle{
			Min: image.Point{8, 3},
			Max: image.Point{12, 6},
		}

		got := call(f, r)
		want := true
		if got != want {
			t.Errorf("SubImage: r=%v: got %t, want %t",
				r, got, want)
		}
	}

	{
		r := image.Rectangle{
			Min: image.Point{9, 3},
			Max: image.Point{12, 6},
		}

		got := call(f, r)
		want := false
		if got != want {
			t.Errorf("SubImage: r=%v: got %t, want %t",
				r, got, want)
		}
	}
}

func BenchmarkAt(b *testing.B) {
	m := New(image.Rect(0, 0, 10, 10), color.Palette{
		color.Transparent,
		color.Opaque,
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.At(4, 5)
	}
}

func BenchmarkSet(b *testing.B) {
	m := New(image.Rect(0, 0, 10, 10), color.Palette{
		color.Transparent,
		color.Opaque,
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.SetColorIndex(4, 5, 1)
	}
}
