package heic

// pooled_decoder.go — reusable WASM decoder (added for batch performance).
//
// The package-level Decode/DecodeThumbnail functions instantiate a fresh WASM
// module on every call (rt.InstantiateModule + mod.Close). For batch workloads
// (e.g. hashing thousands of HEIC files) that per-call instantiation dominates.
//
// Decoder holds one long-lived module instance and reuses it across calls via
// the shared decodeWithModule core, so its output is byte-identical to the
// package-level functions. It is NOT safe for concurrent use — create one
// Decoder per worker goroutine.
//
// This is additive: existing package-level functions are unchanged.

import (
	"context"
	"image"
	"io"

	"github.com/tetratelabs/wazero/api"
)

// Decoder is a reusable, single-threaded HEIC decoder backed by one WASM module
// instance. Call Close when done to free the instance.
type Decoder struct {
	mod    api.Module
	closed bool
}

// NewDecoder creates a decoder with its own reusable WASM module instance.
//
// NewDecoder always uses the WASM backend. On platforms where a dynamic
// libheif is used for the package-level functions, a Decoder still decodes via
// WASM; output is equivalent for perceptual-hash / thumbnail purposes.
func NewDecoder() (*Decoder, error) {
	initOnce()
	ctx := context.Background()
	mod, err := rt.InstantiateModule(ctx, cm, mc.WithName(""))
	if err != nil {
		return nil, err
	}
	return &Decoder{mod: mod}, nil
}

// Close frees the underlying WASM module instance.
func (d *Decoder) Close() error {
	if d.closed {
		return nil
	}
	d.closed = true
	return d.mod.Close(context.Background())
}

// DecodeThumbnail decodes the embedded thumbnail from r using the reused
// instance. Returns ErrNoThumbnail if the file has no embedded thumbnail.
func (d *Decoder) DecodeThumbnail(r io.Reader) (image.Image, error) {
	// copyOut=true: the module's linear memory is overwritten by the next call,
	// so the image must own its pixels.
	img, _, err := decodeWithModule(context.Background(), d.mod, r, false, "decode_thumbnail", true, true)
	return img, err
}

// Decode decodes the primary image from r using the reused instance.
func (d *Decoder) Decode(r io.Reader) (image.Image, error) {
	img, _, err := decodeWithModule(context.Background(), d.mod, r, false, "decode", false, true)
	return img, err
}
