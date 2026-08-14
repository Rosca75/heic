package heic

import (
	"image"
	"io"
)

// Decoder is a reusable HEIC decoder retained for API compatibility with earlier
// releases of this fork. The package now pools WASM module instances internally,
// so a Decoder carries no state of its own and exists only so that existing
// callers keep compiling.
//
// A Decoder always uses the WASM backend, which keeps its output independent of
// whether a dynamic libheif is present. Prefer the package-level Decode and
// DecodeThumbnail functions in new code.
//
// Deprecated: use Decode and DecodeThumbnail directly.
type Decoder struct {
	closed bool
}

// NewDecoder returns a ready-to-use Decoder.
func NewDecoder() (*Decoder, error) {
	return &Decoder{}, nil
}

// Close releases the Decoder. It is safe to call multiple times.
func (d *Decoder) Close() error {
	d.closed = true

	return nil
}

// Decode decodes the primary image from r using the WASM backend.
func (d *Decoder) Decode(r io.Reader) (image.Image, error) {
	img, _, err := decode(r, false)

	return img, err
}

// DecodeThumbnail decodes the embedded thumbnail from r using the WASM backend.
// It returns ErrNoThumbnail if the file contains no embedded thumbnail.
func (d *Decoder) DecodeThumbnail(r io.Reader) (image.Image, error) {
	img, _, err := decodeThumbnail(r, false)

	return img, err
}
