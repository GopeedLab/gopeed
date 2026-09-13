# Extension FFmpeg runtime

`gopeed.runtime.ffmpeg.merge(options)` returns a `ReadableStream<Uint8Array>`.
It merges one video track and one audio track without re-encoding, using
[go-ffmpreg](https://codeberg.org/gruf/go-ffmpreg)'s embedded FFmpeg 5.1.10 and
wazero. No system FFmpeg installation is required. go-ffmpreg and its embedded
GPL-enabled FFmpeg are distributed under GPLv3; see the pinned dependency's
LICENSE, README and build scripts for its source and included components.

## HTTP inputs and one downloadable output

```js
gopeed.events.onResolve(async (ctx) => {
  // Resolve these URLs and headers using the site's API in your extension.
  const video = { url: videoURL, headers: { Referer: pageURL } };
  const audio = { url: audioURL, headers: { Referer: pageURL } };
  const url = await gopeed.runtime.blob.createObjectURL(
    () => gopeed.runtime.ffmpeg.merge({ video, audio }),
    { contentType: "video/mp4" }
  );
  ctx.res = {
    name: "video",
    files: [{ name: "video.mp4", req: { url } }]
  };
});
```

The opener creates a new merge on each open. Output size is unknown and range
support must remain disabled. Do not declare the sum of the input sizes as the
output size. The downloader reads the merged output as one file; FFmpeg errors
propagate through the stream instead of completing a partial download.

## JS inputs, including SABR

Both `video` and `audio` accept either an HTTP descriptor or a
`ReadableStream<Uint8Array>`, including mixed inputs. SABR/UMP decoding belongs
to the extension: pass the resulting media streams, with initialization data,
not the SABR protocol bytes. Keep each selected track's codec/configuration
stable for the duration of a merge.

```js
const output = gopeed.runtime.ffmpeg.merge({
  video: videoStream,
  audio: audioStream,
  signal: abortController.signal,
  args: ["-shortest", "-metadata", "title=Example"]
});
```

Creating the output does not start requests or acquire input readers until the
output is pulled. A stream input is single-use. To retry a Blob-backed task,
its opener must create a new SABR session and fresh input streams. Arbitrary
output byte ranges / resumable merged output are not supported.

## Options

- `video`, `audio`: required inputs; HTTP descriptors have `url` and optional
  `headers` (standard `HeadersInit`). HTTP requests use Gopeed's configured
  proxy. Supply required cookies/authorization explicitly; browser or `fetch`
  cookie jars are not implicitly shared. When `User-Agent` is absent, requests
  use the downloader HTTP protocol's configured UA, captured when the extension
  engine is created, matching fetch and XHR. Explicit UA values (including an
  empty string) take precedence.
- `format`: defaults to `"mp4"`. MP4 output always enables fragmentation
  (`frag_keyframe+empty_moov+default_base_moof`) for non-seekable output. The
  other accepted muxers are `"mkv"`/`"matroska"`, `"webm"`, and `"mpegts"`.
  Codec/container incompatibility is an error, not an automatic conversion.
  Choose the matching filename and Blob content type in the extension.
- `args`: optional array of **output options**, not a shell command or a full
  FFmpeg invocation. Defaults select `0:v:0`, `1:a:0`, and `-c copy`. Additional
  `-map` options are additive. Allowed options are listed in `outputOptions` in
  `runtime.go`: metadata, dispositions, bitstream filters, duration/shortest,
  fragmentation, timestamp/interleaving options and `-strict`. Codec options
  must remain `copy`. Extra inputs, output paths, reports and arbitrary global
  options are rejected. Required MP4 fragmentation is enforced by the host.
- `signal`: optional `AbortSignal`. Aborting or cancelling the output stops
  network requests and native execution, cancels input readers, and removes
  temporary buffers. Await/consume the output inside a live engine or retain
  it through a Blob URL; unowned work ends with the extension engine.

The backend defaults to strict input errors (`-xerror`). Native errors include
the FFmpeg exit code and a bounded stderr tail. No host directories are exposed
to the WASM module; it can access only its two input files and standard output.

## Input behavior and resource limits

HTTP inputs start with a 1 MiB Range GET, reusing the bytes for media reads.
A validated 206 response with a known total size enables seeking; a 200
response is consumed as-is, sequentially. Subsequent ranges verify positions,
lengths, total size and available ETag/Last-Modified validators. Without a
server validator, same-size remote changes cannot be reliably detected.
Content encoding is forced to identity. Authentication/status failures and
malformed range responses are errors. There is no full-file seek fallback.

Sequential JS inputs are drained independently into memory ring buffers that
grow on demand up to **32 MiB per input**, independent of total media size.
No temporary files are used. This does not enable seeking. Full buffers apply
async backpressure until space is available or the operation is cancelled.
Extensions must propagate backpressure through their upstream queues and
network reads. Independent upstream track requests avoid blocking one track
behind another in a shared response. Buffers are released on close.
Blob output writes also wait asynchronously for bounded downstream capacity,
so a slow consumer does not block the extension event loop.

At most two WASM merges execute concurrently per process, with a 512 MiB
linear-memory limit per instance. Compilation is shared and lazy; the upstream
embedded package still decompresses its WASM at process initialization. Native
compilation memory, host buffers and JS allocations are additional to the
linear-memory limit. Playback support depends on the selected codecs, not just
the `.mp4` suffix. Platform execution and performance must be tested on target
devices, especially mobile/32-bit devices.
