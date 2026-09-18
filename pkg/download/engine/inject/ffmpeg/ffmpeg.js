(function () {
  const create = __gopeed_ffmpeg_create;
  const read = __gopeed_ffmpeg_read;
  const start = __gopeed_ffmpeg_start;
  const push = __gopeed_ffmpeg_push;
  const end = __gopeed_ffmpeg_end;
  const cancel = __gopeed_ffmpeg_cancel;
  const active = new Set();
  globalThis.__gopeed_ffmpeg_close_all = () => {
    for (const stop of Array.from(active)) stop(new Error("extension engine closed"));
  };

  function input(value, name, allowStream) {
    if (value && typeof value.getReader === "function") {
      if (!allowStream) throw new TypeError("stream inputs must be created by the inputs callback");
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
    const factory = options.inputs;
    if (factory !== undefined && typeof factory !== "function") throw new TypeError("inputs must be a callback");
    if (factory && (options.video !== undefined || options.audio !== undefined)) throw new TypeError("inputs cannot be combined with video/audio");
    let video = factory ? null : input(options.video, "video", false);
    let audio = factory ? null : input(options.audio, "audio", false);
    const args = options.args === undefined ? [] : options.args;
    if (!Array.isArray(args) || args.some(v => typeof v !== "string")) throw new TypeError("args must be a string array");
    const format = options.format === undefined ? "mp4" : options.format;
    if (typeof format !== "string") throw new TypeError("format must be a string");
    const signal = options.signal;
    const id = create({ format, args });
    let controller, initialized = false, stopped = false;
    const readers = [];
    const lifetime = new AbortController();
    let ownedInputs;
    function discardInputs(inputs, reason) {
      for (const source of new Set([inputs && inputs.video, inputs && inputs.audio])) {
        if (source && typeof source.cancel === "function" && !source.locked) Promise.resolve(source.cancel(reason)).catch(() => {});
      }
    }

    function stop(reason) {
      if (stopped) return;
      stopped = true;
      active.delete(shutdown);
      lifetime.abort();
      discardInputs(ownedInputs, reason);
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
    const output = new ReadableStream({
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
          if (!initialized) {
            initialized = true;
            if (stopped) return;
            if (factory) {
              const inputs = await factory({ signal: lifetime.signal });
              if (stopped) { discardInputs(inputs, new Error("FFmpeg operation cancelled")); return; }
              ownedInputs = inputs;
              video = input(inputs && inputs.video, "video", true);
              audio = input(inputs && inputs.audio, "audio", true);
            }
            if (video.stream && video.stream === audio.stream) throw new TypeError("video and audio must be distinct streams");
            for (const source of [video, audio]) {
              if (source.stream) readers.push(source.stream.getReader());
            }
            start(id, { video: video.request, audio: audio.request, format, args });
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
          if (!stopped) {
            const failure = typeof MessageError === "function" && !(error instanceof MessageError)
              ? new MessageError(String(error && error.message || error)) : error;
            c.error(failure); stop(failure);
          }
        }
      },
      cancel(reason) { stop(reason); }
    }, { highWaterMark: 0 });
    if (globalThis.__gopeed_bind_producer) globalThis.__gopeed_bind_producer(output, id);
    return output;
  }
  globalThis.__gopeed_ffmpeg = Object.freeze({ merge, supportsInputFactory: true, supportsProducerProgress: true });
})();
