package heic

import (
	"bytes"
	_ "embed"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
)

//go:embed testdata/test.heic
var testHeic []byte

//go:embed testdata/test8.heic
var testHeic8 []byte

//go:embed testdata/test12.heic
var testHeic12 []byte

//go:embed testdata/gray.heic
var testGray []byte

//go:embed testdata/anim.heic
var testAnim []byte

func TestDecodeAll(t *testing.T) {
	defer func() { ForceWasmMode = false }()

	for _, wasm := range []bool{false, true} {
		ForceWasmMode = wasm

		h, err := DecodeAll(bytes.NewReader(testAnim))
		if err != nil {
			t.Fatalf("wasm=%v: %v", wasm, err)
		}

		if len(h.Image) != 17 || len(h.Delay) != 17 {
			t.Fatalf("wasm=%v: frames=%d delays=%d, want 17", wasm, len(h.Image), len(h.Delay))
		}

		b := h.Image[0].Bounds()
		if b.Dx() != 176 || b.Dy() != 128 {
			t.Fatalf("wasm=%v: frame dims %dx%d, want 176x128", wasm, b.Dx(), b.Dy())
		}

		for i, d := range h.Delay {
			if d < 0.079 || d > 0.081 {
				t.Fatalf("wasm=%v: delay[%d]=%v, want ~0.08s", wasm, i, d)
			}
		}
	}
}

func TestDecode(t *testing.T) {
	img, _, err := decode(bytes.NewReader(testHeic), false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := writeCloser()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	err = jpeg.Encode(w, img, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecode8(t *testing.T) {
	img, _, err := decode(bytes.NewReader(testHeic8), false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := writeCloser()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	err = jpeg.Encode(w, img, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecode12(t *testing.T) {
	img, _, err := decode(bytes.NewReader(testHeic12), false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := writeCloser()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	err = jpeg.Encode(w, img, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeGray(t *testing.T) {
	img, _, err := decode(bytes.NewReader(testGray), false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := writeCloser()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	err = jpeg.Encode(w, img, nil)
	if err != nil {
		t.Error(err)
	}
}

// TestDecodeThumbnailAbsent checks that a file with no embedded thumbnail
// reports ErrNoThumbnail rather than falling back to the primary image.
func TestDecodeThumbnailAbsent(t *testing.T) {
	defer func() { ForceWasmMode = false }()

	for _, wasm := range []bool{false, true} {
		if !wasm {
			requireDynamic(t)
		}
		ForceWasmMode = wasm

		if _, err := DecodeThumbnail(bytes.NewReader(testHeic)); !errors.Is(err, ErrNoThumbnail) {
			t.Errorf("wasm=%v: got %v, want ErrNoThumbnail", wasm, err)
		}

		if _, err := DecodeThumbnailConfig(bytes.NewReader(testHeic)); !errors.Is(err, ErrNoThumbnail) {
			t.Errorf("wasm=%v: config: got %v, want ErrNoThumbnail", wasm, err)
		}
	}
}

// localSamples returns HEIC files kept in testdata for local testing only; they
// are deliberately not committed, so this skips in CI.
func localSamples(t *testing.T) []string {
	t.Helper()

	files, _ := filepath.Glob("testdata/IMG_*.HEIC")
	if len(files) == 0 {
		t.Skip("no local HEIC samples in testdata")
	}

	return files
}

// TestDecodeThumbnailLocal decodes real embedded thumbnails through both
// backends and requires them to agree on presence and dimensions.
func TestDecodeThumbnailLocal(t *testing.T) {
	defer func() { ForceWasmMode = false }()

	for _, f := range localSamples(t) {
		t.Run(filepath.Base(f), func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}

			ForceWasmMode = true
			img, wErr := DecodeThumbnail(bytes.NewReader(data))
			cfg, cErr := DecodeThumbnailConfig(bytes.NewReader(data))
			ForceWasmMode = false

			if !errors.Is(wErr, cErr) && (wErr == nil) != (cErr == nil) {
				t.Fatalf("wasm: decode err %v disagrees with config err %v", wErr, cErr)
			}

			if wErr == nil {
				if b := img.Bounds(); b.Dx() != cfg.Width || b.Dy() != cfg.Height {
					t.Errorf("wasm: bounds %v disagree with config %dx%d", b, cfg.Width, cfg.Height)
				}
			} else if !errors.Is(wErr, ErrNoThumbnail) {
				t.Fatalf("wasm: %v", wErr)
			}

			if Dynamic() != nil {
				t.Skip("libheif not available")
			}

			_, dCfg, dErr := decodeThumbnailDynamic(bytes.NewReader(data), true)

			// libheif is stricter than the Rust crate and rejects some
			// containers outright. Where it cannot decode the primary image
			// either, the divergence is not specific to thumbnails and there
			// is no parity to assert.
			if errors.Is(dErr, ErrDecode) {
				if _, _, err := decodeDynamic(bytes.NewReader(data), false); err != nil {
					t.Skipf("libheif rejects this container: %v", err)
				}
			}

			if errors.Is(wErr, ErrNoThumbnail) != errors.Is(dErr, ErrNoThumbnail) {
				t.Fatalf("presence mismatch: wasm=%v dynamic=%v", wErr, dErr)
			}
			if dErr != nil {
				if !errors.Is(dErr, ErrNoThumbnail) {
					t.Fatalf("dynamic: %v", dErr)
				}

				return
			}

			if cfg.Width != dCfg.Width || cfg.Height != dCfg.Height {
				t.Errorf("dimension mismatch: wasm %dx%d, dynamic %dx%d",
					cfg.Width, cfg.Height, dCfg.Width, dCfg.Height)
			}
		})
	}
}

var inCI, _ = strconv.ParseBool(os.Getenv("CI"))

func requireDynamic(t testing.TB) {
	if err := Dynamic(); err != nil {
		if inCI {
			t.Fatalf("libheif should be available in CI on %s, but got: %v", runtime.GOOS, err)
		}
		t.Helper()
		t.Skipf("skipping dynamic library test; libheif not available: %v", err)
	}
}

func TestDecodeDynamic(t *testing.T) {
	requireDynamic(t)

	img, _, err := decodeDynamic(bytes.NewReader(testHeic), false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := writeCloser()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	err = jpeg.Encode(w, img, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecode8Dynamic(t *testing.T) {
	requireDynamic(t)

	img, _, err := decodeDynamic(bytes.NewReader(testHeic8), false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := writeCloser()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	err = jpeg.Encode(w, img, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecode12Dynamic(t *testing.T) {
	requireDynamic(t)

	img, _, err := decodeDynamic(bytes.NewReader(testHeic12), false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := writeCloser()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	err = jpeg.Encode(w, img, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeGrayDynamic(t *testing.T) {
	requireDynamic(t)

	img, _, err := decodeDynamic(bytes.NewReader(testGray), false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := writeCloser()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	err = jpeg.Encode(w, img, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestImageDecode(t *testing.T) {
	testBothWays(t, func(t *testing.T) {
		img, _, err := image.Decode(bytes.NewReader(testHeic8))
		if err != nil {
			t.Fatal(err)
		}

		err = jpeg.Encode(io.Discard, img, nil)
		if err != nil {
			t.Error(err)
		}
	})
}

func TestDecodeConfig(t *testing.T) {
	testBothWays(t, func(t *testing.T) {
		cfg, err := DecodeConfig(bytes.NewReader(testHeic8))
		if err != nil {
			t.Fatal(err)
		}

		if cfg.Width != 512 {
			t.Errorf("width: got %d, want %d", cfg.Width, 512)
		}

		if cfg.Height != 512 {
			t.Errorf("height: got %d, want %d", cfg.Height, 512)
		}
	})
}

func TestDecodeSync(t *testing.T) {
	wg := sync.WaitGroup{}
	ch := make(chan bool, 2)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			ch <- true
			defer func() { <-ch; wg.Done() }()

			_, _, err := decode(bytes.NewReader(testHeic8), false)
			if err != nil {
				t.Error(err)
				return
			}
		}()
	}

	wg.Wait()
}

func TestDecodeSyncDynamic(t *testing.T) {
	requireDynamic(t)

	wg := sync.WaitGroup{}
	ch := make(chan bool, 2)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			ch <- true
			defer func() { <-ch; wg.Done() }()

			_, _, err := decodeDynamic(bytes.NewReader(testHeic8), false)
			if err != nil {
				t.Error(err)
				return
			}
		}()
	}

	wg.Wait()
}

// smallChunkReader wraps an io.Reader and limits Read calls to small chunks,
// simulating what an io.Reader passed to image.DecodeConfig might legitimately
// do. (The io.Reader contract allows this.)
type smallChunkReader struct{ io.Reader }

func (r smallChunkReader) Read(p []byte) (int, error) {
	const chunkSize = 128
	if len(p) > chunkSize {
		p = p[:chunkSize]
	}
	return r.Reader.Read(p)
}

func TestDecodeConfigViaImagesPackage(t *testing.T) {
	testBothWays(t, func(t *testing.T) {
		cfg, typ, err := image.DecodeConfig(smallChunkReader{bytes.NewReader(testHeic)})
		if err != nil {
			t.Fatal(err)
		}
		if g, w := cfg.Width, 1346; g != w {
			t.Fatalf("invalid width: got %d, want %d", g, w)
		}
		if g, h := cfg.Height, 1346; g != h {
			t.Fatalf("invalid height: got %d, want %d", g, h)
		}
		if typ != "heic" {
			t.Fatalf("invalid type; got %q; want %q", typ, "heic")
		}
	})
}

// testBothWays runs fn in both wasm mode and dynamic library mode, if possible.
func testBothWays(t *testing.T, fn func(t *testing.T)) {
	t.Run("wasm", func(t *testing.T) {
		was := ForceWasmMode
		ForceWasmMode = true
		t.Cleanup(func() { ForceWasmMode = was })
		fn(t)
	})
	t.Run("dynamic", func(t *testing.T) {
		requireDynamic(t)
		fn(t)
	})
}

func BenchmarkDecode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := decode(bytes.NewReader(testHeic8), false)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkDecodeDynamic(b *testing.B) {
	requireDynamic(b)

	for i := 0; i < b.N; i++ {
		_, _, err := decodeDynamic(bytes.NewReader(testHeic8), false)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkDecodeConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := decode(bytes.NewReader(testHeic8), true)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkDecodeConfigDynamic(b *testing.B) {
	requireDynamic(b)

	for i := 0; i < b.N; i++ {
		_, _, err := decodeDynamic(bytes.NewReader(testHeic8), true)
		if err != nil {
			b.Error(err)
		}
	}
}

type discard struct{}

func (d discard) Close() error {
	return nil
}

func (discard) Write(p []byte) (int, error) {
	return len(p), nil
}

var discardCloser io.WriteCloser = discard{}

func writeCloser(s ...string) (io.WriteCloser, error) {
	if len(s) > 0 {
		f, err := os.Create(s[0])
		if err != nil {
			return nil, err
		}

		return f, nil
	}

	return discardCloser, nil
}
