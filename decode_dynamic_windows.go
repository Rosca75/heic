//go:build windows && !(nodynamic || arm || 386 || mips || mipsle)

package heic

// Windows (msvc/Go on x86-64 Windows) returns small structs via sret.
// Declare the function variables with an extra leading *heifError out-param
// and provide wrappers that call them and return the filled heifError.

var (
	_heifGetVersionNumberMajor                     func() uint32
	_heifGetVersionNumberMinor                     func() uint32
	_heifImageHandleGetNumberOfThumbnails          func(*heifImageHandle) int
	_heifImageHandleGetListOfThumbnailIDs          func(*heifImageHandle, *uint32, int) int
	// sret: heif_error returned via first *heifError argument
	_heifImageHandleGetThumbnail                   func(*heifError, *heifImageHandle, uint32, **heifImageHandle) uintptr
	_heifCheckFiletype                             func(*uint8, uint64) int
	_heifContextAlloc                              func() *heifContext
	_heifContextFree                               func(*heifContext)
	// sret: heif_error via first arg
	_heifContextReadFromMemoryWithoutCopy          func(*heifError, *heifContext, *uint8, uint64, *byte) uintptr
	// sret
	_heifContextGetPrimaryImageHandle              func(*heifError, *heifContext, **heifImageHandle) uintptr
	_heifImageHandleGetWidth                       func(*heifImageHandle) int
	_heifImageHandleGetHeight                      func(*heifImageHandle) int
	_heifImageHandleIsPremultipliedAlpha           func(*heifImageHandle) int
	// sret
	_heifImageHandleGetPreferredDecodingColorspace func(*heifError, *heifImageHandle, *int, *int) uintptr
	_heifImageHandleRelease                        func(*heifImageHandle)
	_heifDecodingOptionsAlloc                      func() *heifDecodingOptions
	_heifDecodingOptionsFree                       func(*heifDecodingOptions)
	// sret
	_heifDecodeImage                               func(*heifError, *heifImageHandle, **heifImage, int, int, *heifDecodingOptions) uintptr
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
	var e heifError
	_heifContextReadFromMemoryWithoutCopy(&e, ctx, &data[0], uint64(len(data)), nil)
	return e
}

func heifContextGetPrimaryImageHandle(ctx *heifContext, handle **heifImageHandle) heifError {
	var e heifError
	_heifContextGetPrimaryImageHandle(&e, ctx, handle)
	return e
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
	var e heifError
	_heifImageHandleGetPreferredDecodingColorspace(&e, handle, colorspace, chroma)
	return e
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
	var e heifError
	_heifDecodeImage(&e, handle, img, colorspace, chroma, options)
	return e
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
	var e heifError
	_heifImageHandleGetThumbnail(&e, h, id, out)
	return e
}
