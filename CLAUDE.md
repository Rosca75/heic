# CLAUDE.md

Project-level guidance for working in this repo. Read on every session.

## What this is

A pure-Go HEIF/HEIC image decoder published as `github.com/Rosca75/heic`. It is a
fork of `github.com/gen2brain/heic`, kept close to upstream so changes can be PR'd
back. It implements `image.Decode` / `image.DecodeConfig` registration plus direct
functions.

Three backends, selected at build time and at runtime:

- **wazero WASM path** (`decode_wazero.go`, build tag `!wasm2go`) — the default.
  The decoder is the **pure-Rust `heic` crate** (`lib/lib.rs`, crate `heic = 0.1.6`)
  compiled to `wasm32-unknown-unknown` and embedded as `lib/heic.wasm.gz`, run
  through the `wazero` interpreter. Works on every platform and GOARCH, zero native
  dependencies.
- **wasm2go path** (`decode_wasm2go.go`, build tag `wasm2go`) — the same Rust WASM
  module transpiled to Go by `wasm2go` into the generated `libheic.go` (~86k lines).
  No interpreter; faster, bigger binary.
- **Dynamic path** (`decode_dynamic.go`) — loads the system `libheif` at runtime via
  `purego`. Preferred when present. Enabled on Linux/macOS/Windows on 64-bit
  non-MIPS/loong64 archs; see the build tag.

`heic.go` dispatches each public function based on a `dynamic` bool set at package
init. Callers force WASM with `ForceWasmMode = true`.

**Note the libheif rewrite**: upstream replaced the old libheif/libde265 C-to-WASM
build with the Rust crate. There is no `lib/heif.c` and no WASI SDK anymore. Any
guidance mentioning `heif.wasm.gz`, `wasi-sdk`, or `-Wl,--export=` is stale.

## Fork additions

Everything else comes from upstream verbatim. The fork carries exactly three things:

1. **Module path** `github.com/Rosca75/heic` in `go.mod`.
2. **`DecodeThumbnail` / `DecodeThumbnailConfig`** — decodes the *real* embedded
   thumbnail, never a downscaled primary. Returns `ErrNoThumbnail` when absent.
   - WASM: `lib/lib.rs` exports `decode_thumbnail`, wrapping the crate's
     `DecoderConfig::decode_thumbnail`.
   - Dynamic: `decodeThumbnailDynamic` in `decode_dynamic.go` via libheif's
     `heif_image_handle_get_thumbnail`.
3. **`Decoder` / `NewDecoder` / `Close` / `(*Decoder).Decode` / `.DecodeThumbnail`**
   (`pooled_decoder.go`) — a thin compatibility shim for `Rosca75/dedup-photos` and
   `Rosca75/geo-photo-tagger`. Upstream now pools WASM modules internally, so the
   fork's old hand-rolled pooling is gone; the type is stateless.

## Build & test

```bash
go test ./...                  # default wazero backend
go test -tags wasm2go ./...    # transpiled backend
go test -tags nodynamic ./...  # WASM only, no libheif
go vet ./...
```

Dynamic tests self-skip when libheif is absent, and **fail** when `CI=true`
(`requireDynamic`). On Ubuntu: `sudo apt install -y libheif1 libheif-dev libheif-examples`.

Cross-compile check (upstream CI does this):
```bash
GOOS=windows GOARCH=amd64 go build ./...
GOOS=linux GOARCH=386 go build ./...
```

## Rebuilding the WASM artifacts

Required whenever `lib/lib.rs` or `lib/Cargo.toml` changes. `make -C lib all`
regenerates **both** `lib/heic.wasm.gz` and `libheic.go`; commit them together.

```bash
rustup target add wasm32-unknown-unknown
go install github.com/ncruces/wasm2go@latest   # needs $(go env GOPATH)/bin on PATH
make -C lib all
```

A new Rust `#[no_mangle]` export needs no Makefile edit — unlike the old C build,
there is no export list. It appears in `libheic.go` as `(*module) X<name>`.

## File map

- `heic.go` — public API, error sentinels, thumbnail status constants, shared
  libheif enum mirrors, `init()` registering the format for several brands.
- `decode_wazero.go` / `decode_wasm2go.go` — the two WASM backends. They must stay
  behaviour-identical; each defines `decode`, `decodeSequence`, `decodeThumbnail`.
- `decode_dynamic.go` — purego backend: symbol binding in `init()`, Go wrappers,
  `decodeDynamic`, `decodeDynamicAll`, `decodeThumbnailDynamic`.
- `decode_dynamic_windows.go` / `decode_dynamic_other.go` — ABI-sensitive wrappers.
  Windows can't return structs through purego, so `heif_error` comes back via an
  sret out-param. **Every `heif_error`-returning symbol needs a wrapper in both.**
- `purego_{darwin,unix,windows,other}.go` — `loadLibrary()` per OS. `_other.go` is
  the disabled-platform fallback and must stub every `*Dynamic` function.
- `lib/lib.rs`, `lib/Cargo.toml`, `lib/Makefile` — the Rust shim and its build.
- `libheic.go` — **generated**, do not hand-edit.
- `exif.go`, `isobmff.go`, `sequence.go` — upstream EXIF and image-sequence support.

## Conventions

- **Rust and both WASM artifacts are locked together.** Change `lib/lib.rs` →
  run `make -C lib all` → commit `lib/heic.wasm.gz` and `libheic.go` in the same commit.
- **The three backends must agree** on presence and dimensions for the same input.
  Concrete image types legitimately differ: the WASM path returns `*image.NRGBA`
  (the Rust crate emits RGBA8); the dynamic path returns `*image.YCbCr`, `*image.Gray`,
  `*image.RGBA` or `*image.NRGBA` depending on the file. Don't assert a type across paths.
- **Adding a WASM export takes three edits**: the `#[no_mangle]` fn in `lib.rs`, the
  `api.Function` field + `ExportedFunction` lookup in `decode_wazero.go`, and the
  `X<name>` call in `decode_wasm2go.go`.
- **Errors at package boundaries** are typed sentinels (`ErrMemRead`, `ErrMemWrite`,
  `ErrDecode`, `ErrNoThumbnail`). Wrap internally with `fmt.Errorf("%s: %w", ...)`;
  return the bare sentinel so callers can `errors.Is`.
- **`image.RegisterFormat` is for the generic `Decode`/`DecodeConfig` pair only.**
  Thumbnails are opt-in API, never registered.
- **Don't commit HEIC sample files.** `testdata/IMG_*.HEIC` are local-only, listed in
  `.git/info/exclude`; `TestDecodeThumbnailLocal` skips when they're absent so CI stays
  green. Fetch more from `github.com/Rosca75/dedup-photos/tree/main/samples`. Upstream's
  small committed fixtures (`test.heic`, `gray.heic`, …) stay as they are.
- **Don't mock libheif.** Use the real library behind `requireDynamic(t)`.

## Gotchas

- **libheif is stricter than the Rust crate.** Some real iPhone files (e.g. one
  referencing a non-existent depth image) are rejected outright by libheif at
  `heif_context_read_from_memory_without_copy`, while the WASM path decodes them
  fine. This affects upstream's primary `decodeDynamic` identically — it is not a
  thumbnail bug. Tests skip parity on such files rather than asserting it.
- **`go vet -tags wasm2go` reports "unreachable code" in `libheic.go`.** Pre-existing
  generator noise; upstream's pristine tree emits more of them. Upstream CI runs no vet.
- **`decodeDynamic(r, configOnly=true)` reads only `heifMaxHeaderSize` (256 KB).**
  Never use it as a "can libheif open this?" probe on large files — it will fail on
  truncation. The thumbnail paths deliberately always `io.ReadAll`, because the
  thumbnail item can live anywhere in the container.
- **32-bit and loong64 builds are WASM-only** (build tag on `decode_dynamic.go`).

## Releases and tags

Tags live on **`main`**. Historic fork tags `v0.1.0`/`v0.2.0`/`v0.3.0` were cut on
feature branches; leave them alone, apps still pin `v0.2.0`. `v0.4.0` — the upstream
Rust/WASM sync — is the first tagged on `main`, and is where releases go from now on.

The fork continues its **own** 0.x sequence, independent of upstream's. Go resolves
`github.com/Rosca75/heic@vX.Y.Z` against this repo's tags only, so a version number
also existing upstream does not block it here. The overlap is nonetheless real in
conversation — upstream is past v0.7.x, so "heic v0.4.0" is ambiguous between the two
repos. Always say which repo you mean.

The `upstream` remote is configured with `tagOpt = --no-tags` so its tags don't
pollute this repo. Keep it that way and never run `git fetch upstream --tags`: it
collides on every shared version number, and a stray upstream tag fetched locally
will block creating the fork's own tag of the same name.

## Upstream

Fork of `github.com/gen2brain/heic`, intended to PR changes back. Keep diffs focused
and match the existing code style. The README's CI/pkg.go.dev badges still point at
upstream on purpose, to keep the diff small.
