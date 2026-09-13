(function () {
  const create = __gopeed_ffmpeg_create;
  const read = __gopeed_ffmpeg_read;
  const push = __gopeed_ffmpeg_push;
  const end = __gopeed_ffmpeg_end;
  const cancel = __gopeed_ffmpeg_cancel;
  const active = new Set();
  globalThis.__gopeed_ffmpeg_close_all = () => {
    for (const stop of Array.from(active)) stop(new Error("extension engine closed"));
  };

  function input(value, name) {
    if (value && typeof value.getReader === "function") {
      if (value.locked) throw new TypeError(name + " stream is locked");
      return { stream: value, request: null };
    }
    if (!value || typeof value.url !== "string" || !/^https?:\/\//i.test(value.url)) {
      throw new TypeError(name + " must be an HTTP request or ReadableStream");
    }
    const headers = {};
    new Headers(value.headers || {}).forEach((v, k) => { headers[k] = v; });
    return { request: { url: value.url, headers } };
  }

  function merge(options) {
    if (!options || typeof options !== "object") throw new TypeError("merge options are required");
    const video = input(options.video, "video");
    const audio = input(options.audio, "audio");
    if (video.stream && video.stream === audio.stream) throw new TypeError("video and audio must be distinct streams");
    const args = options.args === undefined ? [] : options.args;
    if (!Array.isArray(args) || args.some(v => typeof v !== "string")) throw new TypeError("args must be a string array");
    const format = options.format === undefined ? "mp4" : options.format;
    if (typeof format !== "string") throw new TypeError("format must be a string");
    const signal = options.signal;
    let id, controller, stopped = false;
    const readers = [];

    function stop(reason) {
      if (stopped) return;
      stopped = true;
      active.delete(shutdown);
      if (id) cancel(id);
      if (signal) signal.removeEventListener("abort", abort);
      for (const reader of readers) {
        Promise.resolve(reader.cancel(reason)).catch(() => {}).then(() => {
          try { reader.releaseLock(); } catch (_) {}
        });
      }
    }
    function shutdown(reason) {
      controller.error(reason);
      stop(reason);
    }
    function abort() {
      const error = new Error("FFmpeg operation aborted");
      error.name = "AbortError";
      stop(error);
      controller.error(error);
    }
    async function pump(reader, index) {
      try {
        while (!stopped) {
          const result = await reader.read();
          if (stopped) return;
          if (result.done) { end(id, index, ""); return; }
          if (!(result.value instanceof Uint8Array)) throw new TypeError("media streams must contain Uint8Array chunks");
          // Keep individual crossings bounded, even if upstream emits a large chunk.
          for (let offset = 0; offset < result.value.byteLength && !stopped; offset += 256 * 1024) {
            if (!await push(id, index, result.value.subarray(offset, offset + 256 * 1024))) return;
          }
        }
      } catch (error) {
        if (!stopped) {
          end(id, index, String(error && error.message || error));
          controller.error(error);
          stop(error);
        }
      }
    }
    return new ReadableStream({
      start(c) {
        controller = c;
        active.add(shutdown);
        if (signal) {
          if (signal.aborted) abort();
          else signal.addEventListener("abort", abort);
        }
      },
      async pull(c) {
        if (stopped) return;
        try {
          if (!id) {
            // Acquire both readers before starting native work.
            for (const source of [video, audio]) {
              if (source.stream) readers.push(source.stream.getReader());
            }
            id = create({ video: video.request, audio: audio.request, format, args });
            let readerIndex = 0;
            for (const [index, source] of [video, audio].entries()) {
              if (source.stream) void pump(readers[readerIndex++], index);
            }
          }
          const chunk = await read(id);
          if (stopped) return;
          if (chunk === null) { stop(); c.close(); }
          else c.enqueue(new Uint8Array(chunk));
        } catch (error) {
          if (!stopped) { c.error(error); stop(error); }
        }
      },
      cancel(reason) { stop(reason); }
    });
  }
  globalThis.__gopeed_ffmpeg = Object.freeze({ merge });
})();
