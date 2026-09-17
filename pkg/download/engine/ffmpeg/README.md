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

Stream inputs **must** be created by an `inputs({ signal })` callback. It returns
`{ video, audio }`, each an HTTP descriptor or `ReadableStream<Uint8Array>`,
including mixed inputs. Do not start producers before this callback. SABR/UMP decoding belongs
to the extension: pass the resulting media streams, with initialization data,
not the SABR protocol bytes. Keep each selected track's codec/configuration
stable for the duration of a merge.

```js
const output = gopeed.runtime.ffmpeg.merge({
  inputs: async ({ signal }) => {
    const session = await prepareSession({ signal });
    const { videoStream, audioStream, abort } = await session.openStreams();
    if (signal.aborted) { abort(); throw new Error("Cancelled"); }
    signal.addEventListener("abort", abort, { once: true });
    return { video: videoStream, audio: audioStream };
  },
  signal: abortController.signal,
  args: ["-shortest", "-metadata", "title=Example"]
});
```

Creating the output does not start requests or acquire input readers until the
output is pulled. Inputs then download to temporary disk chunks even while the
job waits for a global FFmpeg execution slot. Only merging occupies a slot.
The callback runs once per output; its signal aborts on cancellation, engine
shutdown, failure, and completion. A callback must cooperate with cancellation;
returned streams are cancelled even if the callback finishes after cancellation.
A stream input is single-use. To retry a Blob-backed task,
its opener must create a new SABR session and fresh input streams. Arbitrary
output byte ranges / resumable merged output are not supported.

`gopeed.runtime.ffmpeg.supportsInputFactory === true` identifies the callback API.
`gopeed.runtime.ffmpeg.supportsProducerProgress === true` identifies disk-backed
inputs and automatic Blob production progress. Require the latter before using
one shared SABR session for both tracks; older bounded buffers can deadlock.

## Options

- `inputs`: callback returning `{ video, audio }`, synchronously or asynchronously.
  Required for stream inputs. Cannot be combined with top-level `video`/`audio`.
- `video`, `audio`: HTTP descriptors only when supplied at the top level; both
  are required unless `inputs` is supplied. HTTP descriptors have `url` and optional
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

HTTP and sequential JS inputs are drained into **8 MiB disk chunks** beneath
`TempDir/gopeed-media-<random>`. Range downloads stream each response into as many chunks as
needed, tracking valid byte intervals independently of sparse file length.
Verified HTTP Range inputs can seek, skip missing blocks, and re-fetch deleted
blocks. Sequential inputs cannot seek even though their bytes are on disk.
Consumed chunks are deleted; cancellation, errors and completion remove the
remaining input directories. Disk space is not capped; write failures stop the
merge. Media size does not determine the size of in-memory input queues.

JS writes await disk writes, and Blob output writes await bounded downstream
capacity. Extensions must still propagate backpressure through upstream queues
and network reads. Both tracks can share a SABR response because disk spooling
continues independently of which track FFmpeg currently reads.

When a Blob opener returns the merge stream directly, the runtime automatically
associates its input progress with that Blob. No public progress callback or
extra `createObjectURL` argument is required. Return the merge stream itself;
wrapping it in another stream loses this association. Downloaded bytes count
unique input offsets; speed counts received input bytes, including re-fetches.
The task stays downloading with unknown total size while queued or merging;
once input traffic stops the displayed speed settles to zero. Completion uses
the actual merged output size. Generated output waits are governed by producer
cancellation/errors rather than the HTTP body idle timeout; HTTP media inputs
still enforce a 15-second network idle deadline.

`tempDir` defaults to Go's `os.TempDir()` when omitted. Flutter only supplies
an explicit directory on Android/iOS, using `getTemporaryDirectory()` directly;
no extra `gopeed` parent directory is added. Desktop/server callers can override
it via startup JSON or `--temp-dir`.

Initialization never clears this directory. Media inputs, HTTP prefetch and
extension installation staging create unique `gopeed-media-*`,
`gopeed-prefetch-*`, and `gopeed-extension-*` paths directly beneath it. The
downloader tracks only its own paths and removes remaining ones on normal
shutdown, without scanning or deleting other files in a shared temporary
directory. Resources still clean up on completion/cancellation, and consumed
media chunks are deleted promptly. Forced process termination can leave files
behind; the next startup does not delete them. Persistent HLS resume data and
WebView profiles are unchanged.

At most two WASM merges execute concurrently per process, with a 512 MiB
linear-memory limit per instance. Compilation is shared and lazy; the upstream
embedded package still decompresses its WASM at process initialization. Native
compilation memory, host buffers and JS allocations are additional to the
linear-memory limit. Playback support depends on the selected codecs, not just
the `.mp4` suffix. Platform execution and performance must be tested on target
devices, especially mobile/32-bit devices.
