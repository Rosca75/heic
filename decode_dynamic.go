//go:build (linux || darwin || windows) && !(nodynamic || arm || 386 || mips || mipsle)

package heic

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

func decodeDynamic(r io.Reader, configOnly bool) (image.Image, image.Config, error) {
	var err error
	var cfg image.Config
	var data []byte

	if configOnly {
		data, err = io.ReadAll(io.LimitReader(r, heifMaxHeaderSize))
		if err != nil {
			return nil, cfg, fmt.Errorf("read: %w", err)
		}
	} else {
		data, err = io.ReadAll(r)
		if err != nil {
			return nil, cfg, fmt.Errorf("read: %w", err)
		}
	}

	check := heifCheckFiletype(data)
	if check != heifFiletypeYesSupported {
		return nil, cfg, ErrDecode
	}

	ctx := heifContextAlloc()
	defer heifContextFree(ctx)

	var e heifError

	e = heifContextReadFromMemoryWithoutCopy(ctx, data)
	if e.Code != 0 {
		return nil, cfg, ErrDecode
	}

	handle := new(heifImageHandle)

	e = heifContextGetPrimaryImageHandle(ctx, &handle)
	if e.Code != 0 {
		return nil, cfg, ErrDecode
	}
	defer heifImageHandleRelease(handle)

	cfg.Width = heifImageHandleGetWidth(handle)
	cfg.Height = heifImageHandleGetHeight(handle)

	isPremultiplied := heifImageHandleIsPremultipliedAlpha(handle)

	var colorspace, chroma int
	if versionMajor == 1 && versionMinor >= 17 {
		e = heifImageHandleGetPreferredDecodingColorspace(handle, &colorspace, &chroma)
		if e.Code != 0 {
			return nil, cfg, ErrDecode
		}

		if colorspace == heifColorspaceUndefined || chroma == heifChromaUndefined {
			colorspace = heifColorspaceYCbCr
			chroma = heifChroma420
			cfg.ColorModel = color.YCbCrModel
		}
		if colorspace == heifColorspaceRGB {
			chroma = heifChromaInterleavedRGBA
			if isPremultiplied {
				cfg.ColorModel = color.RGBAModel
			} else {
				cfg.ColorModel = color.NRGBAModel
			}
		}
	} else {
		colorspace = heifColorspaceYCbCr
		chroma = heifChroma420
		cfg.ColorModel = color.YCbCrModel
	}

	if configOnly {
		return nil, cfg, nil
	}

	options := heifDecodingOptionsAlloc()
	options.ConvertHdrTo8bit = 1
	defer heifDecodingOptionsFree(options)

	heifImg := new(heifImage)

	e = heifDecodeImage(handle, &heifImg, colorspace, chroma, options)
	if e.Code != 0 {
		return nil, cfg, ErrDecode
	}

	var img image.Image
	rect := image.Rect(0, 0, cfg.Width, cfg.Height)

	switch colorspace {
	case heifColorspaceYCbCr:
		var subsampleRatio image.YCbCrSubsampleRatio
		switch chroma {
		case heifChroma420:
			subsampleRatio = image.YCbCrSubsampleRatio420
		case heifChroma422:
			subsampleRatio = image.YCbCrSubsampleRatio422
		case heifChroma444:
			subsampleRatio = image.YCbCrSubsampleRatio444
		}

		var yStride, uStride int
		y := heifImageGetPlaneReadonly(heifImg, heifChannelY, &yStride)
		cb := heifImageGetPlaneReadonly(heifImg, heifChannelCb, &uStride)
		cr := heifImageGetPlaneReadonly(heifImg, heifChannelCr, &uStride)

		_, _, _, ch := yCbCrSize(rect, subsampleRatio)
		i0 := yStride * cfg.Height
		i1 := yStride*cfg.Height + 1*uStride*ch
		i2 := yStride*cfg.Height + 2*uStride*ch
		b := make([]byte, i2)

		i := &image.YCbCr{
			Y:              b[:i0:i0],
			Cb:             b[i0:i1:i1],
			Cr:             b[i1:i2:i2],
			SubsampleRatio: subsampleRatio,
			YStride:        yStride,
			CStride:        uStride,
			Rect:           rect,
		}

		copy(i.Y, unsafe.Slice(y, yStride*cfg.Height))
		copy(i.Cb, unsafe.Slice(cb, uStride*ch))
		copy(i.Cr, unsafe.Slice(cr, uStride*ch))

		img = i
	case heifColorspaceMonochrome:
		var stride int
		grayData := heifImageGetPlaneReadonly(heifImg, heifChannelY, &stride)
		size := cfg.Height * stride

		i := &image.Gray{
			Pix:    make([]uint8, size),
			Stride: stride,
			Rect:   rect,
		}

		copy(i.Pix, unsafe.Slice(grayData, size))
		img = i
	case heifColorspaceRGB:
		var stride int
		rgbaData := heifImageGetPlaneReadonly(heifImg, heifChannelInterleaved, &stride)
		size := cfg.Height * stride

		if isPremultiplied {
			i := &image.RGBA{
				Pix:    make([]uint8, size),
				Stride: stride,
				Rect:   rect,
			}

			copy(i.Pix, unsafe.Slice(rgbaData, size))
			img = i
		} else {
			i := &image.NRGBA{
				Pix:    make([]uint8, size),
				Stride: stride,
				Rect:   rect,
			}

			copy(i.Pix, unsafe.Slice(rgbaData, size))
			img = i
		}
	default:
		return nil, cfg, fmt.Errorf("unsupported colorspace %d", colorspace)
	}

	runtime.KeepAlive(data)

	return img, cfg, nil
}

func decodeThumbnailDynamic(r io.Reader, configOnly bool) (image.Image, image.Config, error) {
	var err error
	var cfg image.Config
	var data []byte

	if configOnly {
		data, err = io.ReadAll(io.LimitReader(r, heifMaxHeaderSize))
		if err != nil {
			return nil, cfg, fmt.Errorf("read: %w", err)
		}
	} else {
		data, err = io.ReadAll(r)
		if err != nil {
			return nil, cfg, fmt.Errorf("read: %w", err)
		}
	}

	check := heifCheckFiletype(data)
	if check != heifFiletypeYesSupported {
		return nil, cfg, ErrDecode
	}

	ctx := heifContextAlloc()
	defer heifContextFree(ctx)

	var e heifError

	e = heifContextReadFromMemoryWithoutCopy(ctx, data)
	if e.Code != 0 {
		return nil, cfg, ErrDecode
	}

	handle := new(heifImageHandle)

	e = heifContextGetPrimaryImageHandle(ctx, &handle)
	if e.Code != 0 {
		return nil, cfg, ErrDecode
	}
	defer heifImageHandleRelease(handle)

	n := heifImageHandleGetNumberOfThumbnails(handle)
	if n == 0 {
		return nil, cfg, ErrNoThumbnail
	}

	ids := make([]uint32, 1)
	heifImageHandleGetListOfThumbnailIDs(handle, ids)

	thumb := new(heifImageHandle)
	e = heifImageHandleGetThumbnail(handle, ids[0], &thumb)
	if e.Code != 0 {
		return nil, cfg, ErrDecode
	}
	defer heifImageHandleRelease(thumb)

	cfg.Width = heifImageHandleGetWidth(thumb)
	cfg.Height = heifImageHandleGetHeight(thumb)

	isPremultiplied := heifImageHandleIsPremultipliedAlpha(thumb)

	var colorspace, chroma int
	if versionMajor == 1 && versionMinor >= 17 {
		e = heifImageHandleGetPreferredDecodingColorspace(thumb, &colorspace, &chroma)
		if e.Code != 0 {
			return nil, cfg, ErrDecode
		}

		if colorspace == heifColorspaceUndefined || chroma == heifChromaUndefined {
			colorspace = heifColorspaceYCbCr
			chroma = heifChroma420
			cfg.ColorModel = color.YCbCrModel
		}
		if colorspace == heifColorspaceRGB {
			chroma = heifChromaInterleavedRGBA
			if isPremultiplied {
				cfg.ColorModel = color.RGBAModel
			} else {
				cfg.ColorModel = color.NRGBAModel
			}
		}
	} else {
		colorspace = heifColorspaceYCbCr
		chroma = heifChroma420
		cfg.ColorModel = color.YCbCrModel
	}

	if configOnly {
		return nil, cfg, nil
	}

	options := heifDecodingOptionsAlloc()
	options.ConvertHdrTo8bit = 1
	defer heifDecodingOptionsFree(options)

	heifImg := new(heifImage)

	e = heifDecodeImage(thumb, &heifImg, colorspace, chroma, options)
	if e.Code != 0 {
		return nil, cfg, ErrDecode
	}

	var img image.Image
	rect := image.Rect(0, 0, cfg.Width, cfg.Height)

	switch colorspace {
	case heifColorspaceYCbCr:
		var subsampleRatio image.YCbCrSubsampleRatio
		switch chroma {
		case heifChroma420:
			subsampleRatio = image.YCbCrSubsampleRatio420
		case heifChroma422:
			subsampleRatio = image.YCbCrSubsampleRatio422
		case heifChroma444:
			subsampleRatio = image.YCbCrSubsampleRatio444
		}

		var yStride, uStride int
		y := heifImageGetPlaneReadonly(heifImg, heifChannelY, &yStride)
		cb := heifImageGetPlaneReadonly(heifImg, heifChannelCb, &uStride)
		cr := heifImageGetPlaneReadonly(heifImg, heifChannelCr, &uStride)

		_, _, _, ch := yCbCrSize(rect, subsampleRatio)
		i0 := yStride * cfg.Height
		i1 := yStride*cfg.Height + 1*uStride*ch
		i2 := yStride*cfg.Height + 2*uStride*ch
		b := make([]byte, i2)

		i := &image.YCbCr{
			Y:              b[:i0:i0],
			Cb:             b[i0:i1:i1],
			Cr:             b[i1:i2:i2],
			SubsampleRatio: subsampleRatio,
			YStride:        yStride,
			CStride:        uStride,
			Rect:           rect,
		}

		copy(i.Y, unsafe.Slice(y, yStride*cfg.Height))
		copy(i.Cb, unsafe.Slice(cb, uStride*ch))
		copy(i.Cr, unsafe.Slice(cr, uStride*ch))

		img = i
	case heifColorspaceMonochrome:
		var stride int
		grayData := heifImageGetPlaneReadonly(heifImg, heifChannelY, &stride)
		size := cfg.Height * stride

		i := &image.Gray{
			Pix:    make([]uint8, size),
			Stride: stride,
			Rect:   rect,
		}

		copy(i.Pix, unsafe.Slice(grayData, size))
		img = i
	case heifColorspaceRGB:
		var stride int
		rgbaData := heifImageGetPlaneReadonly(heifImg, heifChannelInterleaved, &stride)
		size := cfg.Height * stride

		if isPremultiplied {
			i := &image.RGBA{
				Pix:    make([]uint8, size),
				Stride: stride,
				Rect:   rect,
			}

			copy(i.Pix, unsafe.Slice(rgbaData, size))
			img = i
		} else {
			i := &image.NRGBA{
				Pix:    make([]uint8, size),
				Stride: stride,
				Rect:   rect,
			}

			copy(i.Pix, unsafe.Slice(rgbaData, size))
			img = i
		}
	default:
		return nil, cfg, fmt.Errorf("unsupported colorspace %d", colorspace)
	}

	runtime.KeepAlive(data)

	return img, cfg, nil
}


func init() {
	var err error
	defer func() {
		if r := recover(); r != nil {
			dynamic = false
			dynamicErr = fmt.Errorf("%v", r)
		}
	}()

	libheif, err = loadLibrary()
	if err == nil {
		dynamic = true
	} else {
		dynamicErr = err
		return
	}

	// Register common symbols. Platform-specific files provide the
	// function-variable declarations with the correct ABI shapes.
	purego.RegisterLibFunc(&_heifGetVersionNumberMajor, libheif, "heif_get_version_number_major")
	purego.RegisterLibFunc(&_heifGetVersionNumberMinor, libheif, "heif_get_version_number_minor")

	versionMajor = heifGetVersionNumberMajor()
	versionMinor = heifGetVersionNumberMinor()

	if versionMajor == 1 && versionMinor >= 17 {
		purego.RegisterLibFunc(&_heifImageHandleGetPreferredDecodingColorspace, libheif, "heif_image_handle_get_preferred_decoding_colorspace")
	}

	purego.RegisterLibFunc(&_heifImageHandleGetNumberOfThumbnails, libheif, "heif_image_handle_get_number_of_thumbnails")
	purego.RegisterLibFunc(&_heifImageHandleGetListOfThumbnailIDs, libheif, "heif_image_handle_get_list_of_thumbnail_IDs")
	purego.RegisterLibFunc(&_heifImageHandleGetThumbnail, libheif, "heif_image_handle_get_thumbnail")

	purego.RegisterLibFunc(&_heifCheckFiletype, libheif, "heif_check_filetype")
	purego.RegisterLibFunc(&_heifContextAlloc, libheif, "heif_context_alloc")
	purego.RegisterLibFunc(&_heifContextFree, libheif, "heif_context_free")
	purego.RegisterLibFunc(&_heifContextReadFromMemoryWithoutCopy, libheif, "heif_context_read_from_memory_without_copy")
	purego.RegisterLibFunc(&_heifContextGetPrimaryImageHandle, libheif, "heif_context_get_primary_image_handle")
	purego.RegisterLibFunc(&_heifImageHandleGetWidth, libheif, "heif_image_handle_get_width")
	purego.RegisterLibFunc(&_heifImageHandleGetHeight, libheif, "heif_image_handle_get_height")
	purego.RegisterLibFunc(&_heifImageHandleIsPremultipliedAlpha, libheif, "heif_image_handle_is_premultiplied_alpha")
	purego.RegisterLibFunc(&_heifImageHandleRelease, libheif, "heif_image_handle_release")
	purego.RegisterLibFunc(&_heifDecodingOptionsAlloc, libheif, "heif_decoding_options_alloc")
	purego.RegisterLibFunc(&_heifDecodingOptionsFree, libheif, "heif_decoding_options_free")
	purego.RegisterLibFunc(&_heifDecodeImage, libheif, "heif_decode_image")
	purego.RegisterLibFunc(&_heifImageGetPlaneReadonly, libheif, "heif_image_get_plane_readonly")
}

var (
	libheif uintptr

	dynamic    bool
	dynamicErr error

	versionMajor int
	versionMinor int
)

// Platform-specific function-variable declarations and small ABI wrappers
// live in decode_dynamic_other.go and decode_dynamic_windows.go so each
// platform can implement the correct calling convention (sret on Windows).

type heifContext struct{}

type heifImageHandle struct{}

type heifImage struct{}

type heifError struct {
	Code    uint32
	Subcode uint32
	Message *int8
}

type heifDecodingOptions struct {
	Version               uint8
	IgnoreTransformations uint8
	StartProgress         *[0]byte
	OnProgress            *[0]byte
	EndProgress           *[0]byte
	ProgressUserData      *byte
	ConvertHdrTo8bit      uint8
	StrictDecoding        uint8
	DecoderId             *int8
}