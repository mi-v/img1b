// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2020 Mikhail Vladimirov
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package png

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/mi-v/img1b"
	"image/color"
	gopng "image/png"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
)

var filenames = []string{
	"basn0g01",
	"basn0g01-30",
	"basn3p01",
	"ftbbn0g01",
}

var filenamesPaletted = []string{
	"basn3p01",
}

func readPNG(filename string) (*img1b.Image, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Decode(f)
}

// An approximation of the sng command-line tool.
func sng(w io.WriteCloser, filename string, png *img1b.Image) {
	defer w.Close()
	bounds := png.Bounds()

	// Write the filename and IHDR.
	io.WriteString(w, "#SNG: from "+filename+".png\nIHDR {\n")
	fmt.Fprintf(w, "    width: %d; height: %d; bitdepth: 1;\n", bounds.Dx(), bounds.Dy())
	io.WriteString(w, "    using color palette;\n")
	io.WriteString(w, "}\n")

	// Write the PLTE and tRNS (if applicable).
	lastAlpha := -1
	io.WriteString(w, "PLTE {\n")
	for i, c := range png.Palette {
		var r, g, b, a uint8
		switch c := c.(type) {
		case color.RGBA:
			r, g, b, a = c.R, c.G, c.B, 0xff
		case color.NRGBA:
			r, g, b, a = c.R, c.G, c.B, c.A
		default:
			panic("unknown palette color type")
		}
		if a != 0xff {
			lastAlpha = i
		}
		fmt.Fprintf(w, "    (%3d,%3d,%3d)     # rgb = (0x%02x,0x%02x,0x%02x)\n", r, g, b, r, g, b)
	}
	io.WriteString(w, "}\n")
	if lastAlpha != -1 {
		io.WriteString(w, "tRNS {\n")
		for i := 0; i <= lastAlpha; i++ {
			_, _, _, a := png.Palette[i].RGBA()
			a >>= 8
			fmt.Fprintf(w, " %d", a)
		}
		io.WriteString(w, "}\n")
	}

	// Write the IMAGE.
	io.WriteString(w, "IMAGE {\n    pixels base64\n")
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			fmt.Fprintf(w, "%d", png.ColorIndexAt(x, y))
		}
		io.WriteString(w, "\n")
	}
	io.WriteString(w, "}\n")
}

func TestReader(t *testing.T) {
	names := filenames
	for _, fn := range names {
		// Read the .png file.
		img, err := readPNG("testdata/pngsuite/" + fn + ".png")
		if err != nil {
			t.Error(fn, err)
			continue
		}

		piper, pipew := io.Pipe()
		pb := bufio.NewScanner(piper)
		go sng(pipew, fn, img)
		defer piper.Close()

		// Read the .sng file.
		sf, err := os.Open("testdata/pngsuite/" + fn + ".sng")
		if err != nil {
			t.Error(fn, err)
			continue
		}
		defer sf.Close()
		sb := bufio.NewScanner(sf)

		// Compare the two, in SNG format, line by line.
		for {
			pdone := !pb.Scan()
			sdone := !sb.Scan()
			if pdone && sdone {
				break
			}
			if pdone || sdone {
				t.Errorf("%s: Different sizes", fn)
				break
			}
			ps := pb.Text()
			ss := sb.Text()

			// Newer versions of the sng command line tool append an optional
			// color name to the RGB tuple. For example:
			//	# rgb = (0xff,0xff,0xff) grey100
			//	# rgb = (0x00,0x00,0xff) blue1
			// instead of the older version's plainer:
			//	# rgb = (0xff,0xff,0xff)
			//	# rgb = (0x00,0x00,0xff)
			// We strip any such name.
			if strings.Contains(ss, "# rgb = (") && !strings.HasSuffix(ss, ")") {
				if i := strings.LastIndex(ss, ") "); i >= 0 {
					ss = ss[:i+1]
				}
			}

			if ps != ss {
				t.Errorf("%s: Mismatch\n%s\nversus\n%s\n", fn, ps, ss)
				break
			}
		}
		if pb.Err() != nil {
			t.Error(fn, pb.Err())
		}
		if sb.Err() != nil {
			t.Error(fn, sb.Err())
		}
	}
}

var readerErrors = []struct {
	file string
	err  string
}{
	{"invalid-zlib.png", "zlib: invalid checksum"},
	{"invalid-crc32.png", "invalid checksum"},
	{"invalid-noend.png", "unexpected EOF"},
	{"invalid-trunc.png", "unexpected EOF"},
}

func TestReaderError(t *testing.T) {
	for _, tt := range readerErrors {
		img, err := readPNG("testdata/" + tt.file)
		if err == nil {
			t.Errorf("decoding %s: missing error", tt.file)
			continue
		}
		if !strings.Contains(err.Error(), tt.err) {
			t.Errorf("decoding %s: %s, want %s", tt.file, err, tt.err)
		}
		if img != nil {
			t.Errorf("decoding %s: have image + error", tt.file)
		}
	}
}

func TestPalettedDecodeConfig(t *testing.T) {
	for _, fn := range filenamesPaletted {
		f, err := os.Open("testdata/pngsuite/" + fn + ".png")
		if err != nil {
			t.Errorf("%s: open failed: %v", fn, err)
			continue
		}
		defer f.Close()
		cfg, err := DecodeConfig(f)
		if err != nil {
			t.Errorf("%s: %v", fn, err)
			continue
		}
		pal, ok := cfg.ColorModel.(color.Palette)
		if !ok {
			t.Errorf("%s: expected paletted color model", fn)
			continue
		}
		if pal == nil {
			t.Errorf("%s: palette not initialized", fn)
			continue
		}
	}
}

func TestInterlaced(t *testing.T) {
	a, err := readPNG("testdata/gradient.png")
	if err != nil {
		t.Fatal(err)
	}
	b, err := readPNG("testdata/gradient.interlaced.png")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("decodings differ:\nnon-interlaced:\n%#v\ninterlaced:\n%#v", a, b)
	}
}

func TestIncompleteIDATOnRowBoundary(t *testing.T) {
	// The following is an invalid 1x2 grayscale PNG image. The header is OK,
	// but the zlib-compressed IDAT payload contains two bytes "\x02\x00",
	// which is only one row of data (the leading "\x02" is a row filter).
	const (
		ihdr = "\x00\x00\x00\x0dIHDR\x00\x00\x00\x01\x00\x00\x00\x02\x01\x00\x00\x00\x00\xb1\xfa\x8b\x8a"
		idat = "\x00\x00\x00\x0eIDAT\x78\x9c\x62\x62\x00\x04\x00\x00\xff\xff\x00\x06\x00\x03\xfa\xd0\x59\xae"
		iend = "\x00\x00\x00\x00IEND\xae\x42\x60\x82"
	)
	_, err := Decode(strings.NewReader(pngHeader + ihdr + idat + iend))
	if err == nil {
		t.Fatal("got nil error, want non-nil")
	}
}

func TestTrailingIDATChunks(t *testing.T) {
	// The following is a valid 1x1 PNG image containing color.Gray{255} and
	// a trailing zero-length IDAT chunk (see PNG specification section 12.9):
	const (
		ihdr      = "\x00\x00\x00\x0dIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x01\x00\x00\x00\x00\x37\x6e\xf9\x24"
		idatWhite = "\x00\x00\x00\x0eIDAT\x78\x9c\x62\xfa\x0f\x08\x00\x00\xff\xff\x01\x05\x01\x02\x5a\xdd\x39\xcd"
		idatZero  = "\x00\x00\x00\x00IDAT\x35\xaf\x06\x1e"
		iend      = "\x00\x00\x00\x00IEND\xae\x42\x60\x82"
	)
	_, err := Decode(strings.NewReader(pngHeader + ihdr + idatWhite + idatZero + iend))
	if err != nil {
		t.Fatalf("decoding valid image: %v", err)
	}

	// Non-zero-length trailing IDAT chunks should be ignored (recoverable error).
	// The following chunk contains a single pixel with color.Gray{0}.
	const idatBlack = "\x00\x00\x00\x0eIDAT\x78\x9c\x62\x62\x00\x04\x00\x00\xff\xff\x00\x06\x00\x03\xfa\xd0\x59\xae"

	img, err := Decode(strings.NewReader(pngHeader + ihdr + idatWhite + idatBlack + iend))
	if err != nil {
		t.Fatalf("trailing IDAT not ignored: %v", err)
	}
	if img.At(0, 0) == (color.Gray{0}) {
		t.Fatal("decoded image from trailing IDAT chunk")
	}
}

func TestMultipletRNSChunks(t *testing.T) {
	/*
		The following is a valid 1x1 paletted PNG image with a 1-element palette
		containing color.NRGBA{0xff, 0x00, 0x00, 0x7f}:
			0000000: 8950 4e47 0d0a 1a0a 0000 000d 4948 4452  .PNG........IHDR
			0000010: 0000 0001 0000 0001 0103 0000 0025 db56  .............%.V
			0000020: ca00 0000 0350 4c54 45ff 0000 19e2 0937  .....PLTE......7
			0000030: 0000 0001 7452 4e53 7f80 5cb4 cb00 0000  ....tRNS..\.....
			0000040: 0e49 4441 5478 9c62 6200 0400 00ff ff00  .IDATx.bb.......
			0000050: 0600 03fa d059 ae00 0000 0049 454e 44ae  .....Y.....IEND.
			0000060: 4260 82                                  B`.
		Dropping the tRNS chunk makes that color's alpha 0xff instead of 0x7f.
	*/
	const (
		ihdr = "\x00\x00\x00\x0dIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x01\x03\x00\x00\x00\x25\xdb\x56\xca"
		plte = "\x00\x00\x00\x03PLTE\xff\x00\x00\x19\xe2\x09\x37"
		trns = "\x00\x00\x00\x01tRNS\x7f\x80\x5c\xb4\xcb"
		idat = "\x00\x00\x00\x0aIDAT\x78\x5e\x63\x62\x00\x00\x00\x06\x00\x03\xa0\x06\x57\x66"
		iend = "\x00\x00\x00\x00IEND\xae\x42\x60\x82"
	)
	for i := 0; i < 4; i++ {
		var b []byte
		b = append(b, pngHeader...)
		b = append(b, ihdr...)
		b = append(b, plte...)
		for j := 0; j < i; j++ {
			b = append(b, trns...)
		}
		b = append(b, idat...)
		b = append(b, iend...)

		var want color.Color
		m, err := Decode(bytes.NewReader(b))
		switch i {
		case 0:
			if err != nil {
				t.Errorf("%d tRNS chunks: %v", i, err)
				continue
			}
			want = color.RGBA{0xff, 0x00, 0x00, 0xff}
		case 1:
			if err != nil {
				t.Errorf("%d tRNS chunks: %v", i, err)
				continue
			}
			want = color.NRGBA{0xff, 0x00, 0x00, 0x7f}
		default:
			if err == nil {
				t.Errorf("%d tRNS chunks: got nil error, want non-nil", i)
			}
			continue
		}
		if got := m.At(0, 0); got != want {
			t.Errorf("%d tRNS chunks: got %T %v, want %T %v", i, got, got, want, want)
		}
	}
}

func TestUnknownChunkLengthUnderflow(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x06, 0xf4, 0x7c, 0x55, 0x04, 0x1a,
		0xd3, 0x11, 0x9a, 0x73, 0x00, 0x00, 0xf8, 0x1e, 0xf3, 0x2e, 0x00, 0x00,
		0x01, 0x00, 0xff, 0xff, 0xff, 0xff, 0x07, 0xf4, 0x7c, 0x55, 0x04, 0x1a,
		0xd3}
	_, err := Decode(bytes.NewReader(data))
	if err == nil {
		t.Errorf("Didn't fail reading an unknown chunk with length 0xffffffff")
	}
}

func TestOutOfPalettePixel(t *testing.T) {
	// IDAT contains a reference to a palette index that does not exist in the file.
	data := []byte{
		// png header
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		// IHDR
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00,
		0x00, 0x00, 0x01, 0x01, 0x03, 0x00, 0x00, 0x00, 0x25, 0xdb, 0x56, 0xca,
		// PLTE
		0x00, 0x00, 0x00, 0x03, 0x50, 0x4c, 0x54, 0x45, 0xff, 0x00, 0x00, 0x19, 0xe2,
		0x09, 0x37,
		// IDAT
		0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x5e, 0x63, 0x6a, 0x00,
		0x00, 0x00, 0x86, 0x00, 0x83, 0x9f, 0x64, 0x81, 0xa1,
		// IEND
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
	img, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Errorf("decoding invalid palette png: unexpected error %v", err)
		return
	}

	// Expect that the palette is extended with opaque black.
	want := color.RGBA{0x00, 0x00, 0x00, 0xff}
	if got := img.At(0, 0); got != want {
		t.Errorf("got %T %v, expected %T %v", got, got, want, want)
	}
}

func BenchmarkDecode(b *testing.B) {
	data, err := ioutil.ReadFile("testdata/benchBW.png")
	if err != nil {
		b.Fatal(err)
	}
	cfg, err := DecodeConfig(bytes.NewReader(data))
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(cfg.Width * cfg.Height / 8))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Decode(bytes.NewReader(data))
	}
}

func BenchmarkDecodeStock(b *testing.B) {
	data, err := ioutil.ReadFile("testdata/benchBW.png")
	if err != nil {
		b.Fatal(err)
	}
	cfg, err := DecodeConfig(bytes.NewReader(data))
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(cfg.Width * cfg.Height))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gopng.Decode(bytes.NewReader(data))
	}
}
