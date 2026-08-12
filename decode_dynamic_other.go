//go:build (linux || darwin) && !(nodynamic || arm || 386 || mips || mipsle)

package heic

import (
	"unsafe"
)

var (
	_heifGetVersionNumberMajor                     func() uint32
	_heifGetVersionNumberMinor                     func() uint32
	_heifImageHandleGetNumberOfThumbnails          func(*heifImageHandle) int
	_heifImageHandleGetListOfThumbnailIDs          func(*heifImageHandle, *uint32, int) int
	_heifImageHandleGetThumbnail                   func(*heifImageHandle, uint32, **heifImageHandle) uintptr
	_heifCheckFiletype                             func(*uint8, uint64) int
	_heifContextAlloc                              func() *heifContext
	_heifContextFree                               func(*heifContext)
	_heifContextReadFromMemoryWithoutCopy          func(*heifContext, *uint8, uint64, *byte) uintptr
	_heifContextGetPrimaryImageHandle              func(*heifContext, **heifImageHandle) uintptr
	_heifImageHandleGetWidth                       func(*heifImageHandle) int
	_heifImageHandleGetHeight                      func(*heifImageHandle) int
	_heifImageHandleIsPremultipliedAlpha           func(*heifImageHandle) int
	_heifImageHandleGetPreferredDecodingColorspace func(*heifImageHandle, *int, *int) uintptr
	_heifImageHandleRelease                        func(*heifImageHandle)
	_heifDecodingOptionsAlloc                      func() *heifDecodingOptions
	_heifDecodingOptionsFree                       func(*heifDecodingOptions)
	_heifDecodeImage                               func(*heifImageHandle, **heifImage, int, int, *heifDecodingOptions) uintptr
	_heifImageGetPlaneReadonly                     func(*heifImage, int, *int) *uint8
)

func heifGetVersionNumberMajor() int {
	return int(_heifGetVersionNumberMajor())
}

func heifGetVersionNumberMinor() int {
	return int(_heifGetVersionNumberMinor())
}

func heifCheckFiletype(data []byte) int {
	return _heifCheckFiletype(&data[0], uint64(len(data)))
}

func heifContextAlloc() *heifContext {
	return _heifContextAlloc()
}

func heifContextFree(ctx *heifContext) {
	_heifContextFree(ctx)
}

func heifContextReadFromMemoryWithoutCopy(ctx *heifContext, data []byte) heifError {
	ret := _heifContextReadFromMemoryWithoutCopy(ctx, &data[0], uint64(len(data)), nil)

	return *(*heifError)(unsafe.Pointer(&ret))
}

func heifContextGetPrimaryImageHandle(ctx *heifContext, handle **heifImageHandle) heifError {
	ret := _heifContextGetPrimaryImageHandle(ctx, handle)

	return *(*heifError)(unsafe.Pointer(&ret))
}

func heifImageHandleGetWidth(handle *heifImageHandle) int {
	return _heifImageHandleGetWidth(handle)
}

func heifImageHandleGetHeight(handle *heifImageHandle) int {
	return _heifImageHandleGetHeight(handle)
}

func heifImageHandleIsPremultipliedAlpha(handle *heifImageHandle) bool {
	ret := _heifImageHandleIsPremultipliedAlpha(handle)

	return ret != 0
}

func heifImageHandleGetPreferredDecodingColorspace(handle *heifImageHandle, colorspace *int, chroma *int) heifError {
	ret := _heifImageHandleGetPreferredDecodingColorspace(handle, colorspace, chroma)

	return *(*heifError)(unsafe.Pointer(&ret))
}

func heifImageHandleRelease(handle *heifImageHandle) {
	_heifImageHandleRelease(handle)
}

func heifDecodingOptionsAlloc() *heifDecodingOptions {
	return _heifDecodingOptionsAlloc()
}

func heifDecodingOptionsFree(options *heifDecodingOptions) {
	_heifDecodingOptionsFree(options)
}

func heifDecodeImage(handle *heifImageHandle, img **heifImage, colorspace int, chroma int, options *heifDecodingOptions) heifError {
	ret := _heifDecodeImage(handle, img, colorspace, chroma, options)

	return *(*heifError)(unsafe.Pointer(&ret))
}

func heifImageGetPlaneReadonly(img *heifImage, channel int, stride *int) *uint8 {
	return _heifImageGetPlaneReadonly(img, channel, stride)
}

func heifImageHandleGetNumberOfThumbnails(h *heifImageHandle) int {
	return _heifImageHandleGetNumberOfThumbnails(h)
}

func heifImageHandleGetListOfThumbnailIDs(h *heifImageHandle, ids []uint32) int {
	return _heifImageHandleGetListOfThumbnailIDs(h, &ids[0], len(ids))
}

func heifImageHandleGetThumbnail(h *heifImageHandle, id uint32, out **heifImageHandle) heifError {
	ret := _heifImageHandleGetThumbnail(h, id, out)
	return *(*heifError)(unsafe.Pointer(&ret))
}
