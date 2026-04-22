(function () {
  const createBlobObjectURL = globalThis.__gopeed_create_blob_object_url;
  const createWritableObjectURL = globalThis.__gopeed_create_writable_stream_object_url;
  const writeWritableObjectURL = globalThis.__gopeed_write_writable_stream_object_url;
  const closeWritableObjectURL = globalThis.__gopeed_close_writable_stream_object_url;
  const abortWritableObjectURL = globalThis.__gopeed_abort_writable_stream_object_url;
  const revokeObjectURL = globalThis.__gopeed_revoke_object_url;
  const fetchOpen = globalThis.__gopeed_fetch_open;
  const fetchRead = globalThis.__gopeed_fetch_read;
  const fetchClose = globalThis.__gopeed_fetch_close;
  const fetchAbort = globalThis.__gopeed_fetch_abort;

  if (typeof globalThis.ReadableStream === "undefined") {
    class ReadableStreamDefaultController {
      constructor(stream) {
        this._stream = stream;
      }

      enqueue(chunk) {
        this._stream._enqueue(chunk);
      }

      close() {
        this._stream._close();
      }

      error(err) {
        this._stream._error(err);
      }
    }

    class ReadableStreamDefaultReader {
      constructor(stream) {
        this._stream = stream;
      }

      read() {
        return this._stream._read();
      }

      cancel(reason) {
        if (!this._stream) {
          return Promise.resolve();
        }
        return this._stream.cancel(reason);
      }

      releaseLock() {
        if (this._stream) {
          this._stream._reader = null;
          this._stream.locked = false;
          this._stream = null;
        }
      }
    }

    class ReadableStream {
      constructor(source = {}) {
        this._source = source;
        this._queue = [];
        this._waiters = [];
        this._closed = false;
        this._errored = null;
        this._reader = null;
        this._pulling = false;
        this.locked = false;
        this._controller = new ReadableStreamDefaultController(this);
        if (typeof source.start === "function") {
          source.start(this._controller);
        }
      }

      _enqueue(chunk) {
        if (this._closed || this._errored) {
          return;
        }
        if (this._waiters.length > 0) {
          const waiter = this._waiters.shift();
          waiter.resolve({ done: false, value: chunk });
          return;
        }
        this._queue.push(chunk);
      }

      _close() {
        this._closed = true;
        while (this._waiters.length > 0) {
          this._waiters.shift().resolve({ done: true, value: undefined });
        }
      }

      _error(err) {
        this._errored = err || new Error("ReadableStream error");
        while (this._waiters.length > 0) {
          this._waiters.shift().reject(this._errored);
        }
      }

      _read() {
        if (this._errored) {
          return Promise.reject(this._errored);
        }
        this._markBodyUsed();
        if (this._queue.length > 0) {
          return Promise.resolve({ done: false, value: this._queue.shift() });
        }
        if (this._closed) {
          return Promise.resolve({ done: true, value: undefined });
        }
        this._maybePull();
        if (this._queue.length > 0) {
          return Promise.resolve({ done: false, value: this._queue.shift() });
        }
        if (this._closed) {
          return Promise.resolve({ done: true, value: undefined });
        }
        return new Promise((resolve, reject) => {
          this._waiters.push({ resolve, reject });
          this._maybePull();
        });
      }

      _maybePull() {
        if (this._closed || this._errored || this._pulling) {
          return;
        }
        if (this._queue.length > 0) {
          return;
        }
        if (!this._source || typeof this._source.pull !== "function") {
          return;
        }
        this._pulling = true;
        Promise.resolve(this._source.pull(this._controller))
          .catch((err) => {
            this._error(err);
          })
          .finally(() => {
            this._pulling = false;
            if (!this._closed && !this._errored && this._queue.length === 0 && this._waiters.length > 0) {
              this._maybePull();
            }
          });
      }

      _markBodyUsed() {
        if (typeof this.__gopeedMarkBodyUsed === "function") {
          this.__gopeedMarkBodyUsed();
        }
      }

      getReader() {
        if (this.locked) {
          throw new TypeError("ReadableStream is locked");
        }
        this.locked = true;
        this._reader = new ReadableStreamDefaultReader(this);
        return this._reader;
      }

      cancel(reason) {
        this._markBodyUsed();
        if (this._source && typeof this._source.cancel === "function") {
          return Promise.resolve(this._source.cancel(reason)).then(() => {
            this._close();
          });
        }
        this._close();
        return Promise.resolve();
      }

      async pipeTo(dest) {
        const reader = this.getReader();
        const writer = dest.getWriter();
        try {
          while (true) {
            const { done, value } = await reader.read();
            if (done) {
              await writer.close();
              break;
            }
            await writer.write(value);
          }
        } catch (err) {
          await writer.abort(err);
          throw err;
        } finally {
          reader.releaseLock();
          writer.releaseLock();
        }
      }
    }

    globalThis.ReadableStream = ReadableStream;
  }

  if (typeof globalThis.WritableStream === "undefined") {
    class WritableStreamDefaultWriter {
      constructor(stream) {
        if (stream.locked) {
          throw new TypeError("WritableStream is locked");
        }
        this._stream = stream;
        this._stream.locked = true;
      }

      write(chunk) {
        return this._stream._write(chunk);
      }

      close() {
        return this._stream._close();
      }

      abort(reason) {
        return this._stream._abort(reason);
      }

      releaseLock() {
        if (this._stream) {
          this._stream.locked = false;
          this._stream = null;
        }
      }
    }

    class WritableStream {
      constructor(sink = {}) {
        this._sink = sink;
        this._observers = [];
        this._state = "writable";
        this.locked = false;
      }

      _targets() {
        return [this._sink, ...this._observers];
      }

      _write(chunk) {
        if (this._state !== "writable") {
          return Promise.reject(new TypeError("WritableStream is not writable"));
        }
        let chain = Promise.resolve();
        for (const target of this._targets()) {
          if (target && typeof target.write === "function") {
            chain = chain.then(() => target.write(chunk));
          }
        }
        return chain;
      }

      _close() {
        if (this._state !== "writable") {
          return Promise.reject(new TypeError("WritableStream is not writable"));
        }
        this._state = "closed";
        let chain = Promise.resolve();
        for (const target of this._targets()) {
          if (target && typeof target.close === "function") {
            chain = chain.then(() => target.close());
          }
        }
        return chain;
      }

      _abort(reason) {
        if (this._state === "errored") {
          return Promise.resolve();
        }
        this._state = "errored";
        let chain = Promise.resolve();
        for (const target of this._targets()) {
          if (target && typeof target.abort === "function") {
            chain = chain.then(() => target.abort(reason));
          }
        }
        return chain;
      }

      _addObserver(observer) {
        this._observers.push(observer);
      }

      getWriter() {
        return new WritableStreamDefaultWriter(this);
      }
    }

    globalThis.WritableStream = WritableStream;
  }

  if (typeof globalThis.TransformStream === "undefined") {
    class TransformStream {
      constructor(transformer = {}) {
        let readableController;
        this.readable = new ReadableStream({
          start(controller) {
            readableController = controller;
          }
        });
        this.writable = new WritableStream({
          async write(chunk) {
            if (typeof transformer.transform === "function") {
              await transformer.transform(chunk, readableController);
              return;
            }
            readableController.enqueue(chunk);
          },
          async close() {
            if (typeof transformer.flush === "function") {
              await transformer.flush(readableController);
            }
            readableController.close();
          },
          async abort(reason) {
            if (typeof transformer.abort === "function") {
              await transformer.abort(reason);
            }
            readableController.error(reason || new Error("TransformStream aborted"));
          }
        });
      }
    }

    globalThis.TransformStream = TransformStream;
  }

  if (typeof globalThis.ByteLengthQueuingStrategy === "undefined") {
    globalThis.ByteLengthQueuingStrategy = class ByteLengthQueuingStrategy {
      constructor({ highWaterMark }) {
        this.highWaterMark = highWaterMark;
      }

      size(chunk) {
        if (typeof chunk === "string") {
          return chunk.length;
        }
        if (chunk && typeof chunk.byteLength === "number") {
          return chunk.byteLength;
        }
        return 1;
      }
    };
  }

  if (typeof globalThis.CountQueuingStrategy === "undefined") {
    globalThis.CountQueuingStrategy = class CountQueuingStrategy {
      constructor({ highWaterMark }) {
        this.highWaterMark = highWaterMark;
      }

      size() {
        return 1;
      }
    };
  }

  function toUint8Array(chunk) {
    if (chunk == null) {
      return new Uint8Array(0);
    }
    if (chunk instanceof Uint8Array) {
      return chunk;
    }
    if (typeof chunk === "string") {
      return new TextEncoder().encode(chunk);
    }
    if (chunk instanceof ArrayBuffer) {
      return new Uint8Array(chunk);
    }
    if (typeof ArrayBuffer !== "undefined" && ArrayBuffer.isView && ArrayBuffer.isView(chunk)) {
      return new Uint8Array(chunk.buffer, chunk.byteOffset, chunk.byteLength);
    }
    if (typeof Blob !== "undefined" && chunk instanceof Blob) {
      if (chunk._buffer instanceof Uint8Array) {
        return chunk._buffer;
      }
    }
    return new Uint8Array(0);
  }

  function createBodyReadableStream(owner) {
    return new ReadableStream({
      start(controller) {
        Promise.resolve().then(async () => {
          owner.bodyUsed = true;
          if (owner._noBody) {
            controller.close();
            return;
          }
          if (owner._bodyArrayBuffer) {
            controller.enqueue(toUint8Array(owner._bodyArrayBuffer));
            controller.close();
            return;
          }
          if (owner._bodyBlob) {
            const data = await owner._bodyBlob.arrayBuffer();
            controller.enqueue(new Uint8Array(data));
            controller.close();
            return;
          }
          if (owner._bodyText != null) {
            controller.enqueue(toUint8Array(owner._bodyText));
            controller.close();
            return;
          }
          if (owner._bodyInit != null) {
            controller.enqueue(toUint8Array(owner._bodyInit));
          }
          controller.close();
        }).catch((err) => {
          controller.error(err);
        });
      }
    });
  }

  async function readAllFromStream(stream, asText) {
    if (!stream) {
      return asText ? "" : new Uint8Array(0);
    }
    const reader = stream.getReader();
    const chunks = [];
    let total = 0;
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) {
          break;
        }
        const chunk = toUint8Array(value);
        chunks.push(chunk);
        total += chunk.byteLength;
      }
    } finally {
      reader.releaseLock();
    }
    const merged = new Uint8Array(total);
    let offset = 0;
    for (const chunk of chunks) {
      merged.set(chunk, offset);
      offset += chunk.byteLength;
    }
    if (asText) {
      return new TextDecoder().decode(merged);
    }
    return merged;
  }

  function attachResponseStreaming(response, stream) {
    response.__gopeedBodyStream = stream;
    response.__gopeedBodyConsumed = false;
    const ensureUnused = () => {
      if (response.__gopeedBodyConsumed) {
        throw new TypeError("Already read");
      }
    };
    const markBodyUsed = () => {
      ensureUnused();
      response.__gopeedBodyConsumed = true;
      response.bodyUsed = true;
    };
    if (stream) {
      stream.__gopeedMarkBodyUsed = () => {
        if (!response.__gopeedBodyConsumed) {
          response.__gopeedBodyConsumed = true;
          response.bodyUsed = true;
        }
      };
    }
    response.text = async function () {
      ensureUnused();
      markBodyUsed();
      return readAllFromStream(stream, true);
    };
    response.arrayBuffer = async function () {
      ensureUnused();
      markBodyUsed();
      const bytes = await readAllFromStream(stream, false);
      return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength);
    };
    response.blob = async function () {
      ensureUnused();
      markBodyUsed();
      const bytes = await readAllFromStream(stream, false);
      const contentType = this.headers && this.headers.get ? (this.headers.get("content-type") || "") : "";
      return new Blob([bytes], { type: contentType });
    };
    response.json = async function () {
      const text = await this.text();
      return JSON.parse(text);
    };
    return response;
  }

  if (typeof globalThis.Response === "function") {
    const responseProto = globalThis.Response.prototype;
    const bodyDescriptor = Object.getOwnPropertyDescriptor(responseProto, "body");
    if (!bodyDescriptor || typeof bodyDescriptor.get !== "function") {
      Object.defineProperty(responseProto, "body", {
        configurable: true,
        enumerable: true,
        get() {
          if (this.__gopeedBodyStream) {
            return this.__gopeedBodyStream;
          }
          if (!this.__gopeedBodyStream) {
            this.__gopeedBodyStream = createBodyReadableStream(this);
          }
          return this.__gopeedBodyStream;
        }
      });
    }
  }

  const originalCreateObjectURL = typeof URL.createObjectURL === "function"
    ? URL.createObjectURL.bind(URL)
    : null;
  const originalRevokeObjectURL = typeof URL.revokeObjectURL === "function"
    ? URL.revokeObjectURL.bind(URL)
    : null;
  const resumableReadableObjectURLs = new Map();
  const activeReadableObjectURLs = new Map();
  const readableObjectURLYieldBytes = 256 * 1024;

  function isIgnorableGBlobObjectURLError(error) {
    const message = error == null ? "" : String(error && error.message ? error.message : error);
    return message.indexOf("gblob source revoked") >= 0 ||
      message.indexOf("gblob source not found") >= 0 ||
      message.indexOf("gblob source closed") >= 0 ||
      message.indexOf("gblob source aborted") >= 0;
  }

  function getValueTypeName(value) {
    if (value === null) {
      return "null";
    }
    if (value === undefined) {
      return "undefined";
    }
    if (value && value.constructor && typeof value.constructor.name === "string" && value.constructor.name) {
      return value.constructor.name;
    }
    return typeof value;
  }

  function toReadableStreamReader(value, sourceLabel) {
    if (value instanceof ReadableStream) {
      return value.getReader();
    }
    throw new TypeError(sourceLabel + " must return a ReadableStream, got " + getValueTypeName(value));
  }

  function describeObjectURLValue(value) {
    if (value instanceof Blob) {
      return {
        kind: "blob",
        value,
      };
    }
    if (value instanceof ReadableStream) {
      return {
        kind: "readable",
        initialReadable: value,
        openReadable: null,
        sourceLabel: "URL.createObjectURL ReadableStream",
      };
    }
    if (typeof value === "function") {
      return {
        kind: "opener",
        initialReadable: null,
        openReadable: value,
        sourceLabel: "URL.createObjectURL opener function",
      };
    }
    return {
      kind: "other",
      value,
    };
  }

  function startReadableObjectURL(url, state) {
    activeReadableObjectURLs.set(url, state);
    setTimeout(() => {
      void pumpReadableObjectURL(url, state);
    }, 0);
  }

  function releaseReader(reader, reason) {
    if (!reader) {
      return;
    }
    try {
      if (reason !== undefined && typeof reader.cancel === "function") {
        const result = reader.cancel(reason);
        if (result && typeof result.catch === "function") {
          result.catch(function () {});
        }
      }
    } catch (_) {
    }
    if (typeof reader.releaseLock === "function") {
      try {
        reader.releaseLock();
      } catch (_) {
      }
    }
  }

  function releaseActiveReadable(url, reason) {
    const active = activeReadableObjectURLs.get(url);
    if (!active) {
      return;
    }
    activeReadableObjectURLs.delete(url);
    active.cancelled = true;
    releaseReader(active.reader, reason);
  }

  async function pumpReadableObjectURL(url, state) {
    let reader = null;
    try {
      let source;
      if (state.initialReadable) {
        source = state.initialReadable;
        state.initialReadable = null;
      } else {
        if (typeof state.openReadable !== "function") {
          throw new Error("gblob readable stream is not reopenable");
        }
        source = await state.openReadable(state.offset);
      }
      reader = toReadableStreamReader(source, state.sourceLabel);
      if (activeReadableObjectURLs.get(url) !== state || state.cancelled) {
        releaseReader(reader, "stale gblob producer");
        reader = null;
        return;
      }
      state.reader = reader;
      let bytesSinceYield = 0;
      while (!state.cancelled) {
        const current = activeReadableObjectURLs.get(url);
        if (current !== state) {
          releaseReader(reader, "stale gblob producer");
          reader = null;
          return;
        }
        const { done, value } = await reader.read();
        if (done) {
          if (activeReadableObjectURLs.get(url) === state) {
            activeReadableObjectURLs.delete(url);
            await closeWritableObjectURL(url);
          }
          releaseReader(reader);
          reader = null;
          return;
        }
        const chunk = value instanceof Uint8Array ? value : new Uint8Array(value);
        if (activeReadableObjectURLs.get(url) !== state || state.cancelled) {
          releaseReader(reader, "stale gblob producer");
          reader = null;
          return;
        }
        await writeWritableObjectURL(url, chunk);
        bytesSinceYield += chunk.byteLength;
        if (bytesSinceYield >= readableObjectURLYieldBytes) {
          bytesSinceYield = 0;
          await new Promise((resolve) => setTimeout(resolve, 0));
        }
      }
      releaseReader(reader, "gblob producer cancelled");
      reader = null;
    } catch (error) {
      if (isIgnorableGBlobObjectURLError(error)) {
        if (activeReadableObjectURLs.get(url) === state) {
          activeReadableObjectURLs.delete(url);
        }
        if (reader) {
          releaseReader(reader, "gblob source closed");
          reader = null;
        }
        return;
      }
      if (activeReadableObjectURLs.get(url) === state) {
        activeReadableObjectURLs.delete(url);
        try {
          await abortWritableObjectURL(url, error == null ? "" : String(error && error.message ? error.message : error));
        } catch (_) {
        }
      }
      if (reader) {
        releaseReader(reader, error);
        reader = null;
      }
    }
  }

  globalThis.__gopeed_open_writable_stream_object_url = function (url, offset) {
    const openReadable = resumableReadableObjectURLs.get(url);
    if (typeof openReadable !== "function") {
      throw new Error("gblob resumable opener not found");
    }
    releaseActiveReadable(url, "gblob source reopened");
    startReadableObjectURL(url, {
      initialReadable: null,
      openReadable,
      sourceLabel: "URL.createObjectURL opener function",
      offset: Number(offset) || 0,
      reader: null,
      cancelled: false,
    });
  };

  URL.createObjectURL = function (value) {
    const described = describeObjectURLValue(value);
    if (described.kind === "blob") {
      return createBlobObjectURL(described.value._buffer, described.value.type || "");
    }
    if (described.kind === "readable") {
      const url = createWritableObjectURL(false);
      startReadableObjectURL(url, {
        initialReadable: described.initialReadable,
        openReadable: null,
        sourceLabel: described.sourceLabel,
        offset: 0,
        reader: null,
        cancelled: false,
      });
      return url;
    }
    if (described.kind === "opener") {
      const url = createWritableObjectURL(true);
      resumableReadableObjectURLs.set(url, described.openReadable);
      return url;
    }
    throw new TypeError("Unsupported object type for URL.createObjectURL: " + getValueTypeName(described.value) + ". Expected Blob, ReadableStream, or opener function");
  };

  URL.revokeObjectURL = function (url) {
    if (typeof url === "string" && url.indexOf("gblob:") === 0) {
      resumableReadableObjectURLs.delete(url);
      releaseActiveReadable(url, "gblob source revoked");
      revokeObjectURL(url);
      return;
    }
    if (originalRevokeObjectURL) {
      originalRevokeObjectURL(url);
    }
  };

  const originalFetch = typeof globalThis.fetch === "function"
    ? globalThis.fetch.bind(globalThis)
    : null;

  if (typeof fetchOpen === "function") {
    globalThis.fetch = async function (input, init) {
      const request = new Request(input, init);
      if (originalFetch && (request.method === "HEAD" || request.redirect === "manual" || request.redirect === "error")) {
        return originalFetch(input, init);
      }
      let body = null;
      if (request._bodyFormData) {
        body = request._bodyFormData;
      } else if (request._bodyArrayBuffer) {
        body = request._bodyArrayBuffer;
      } else if (request._bodyBlob) {
        body = await request._bodyBlob.arrayBuffer();
      } else if (request._bodyInit != null && typeof request._bodyInit === "object") {
        body = request._bodyInit;
      } else if (request._bodyText != null) {
        body = request._bodyText;
      } else if (request._bodyInit != null) {
        body = request._bodyInit;
      }
      const headers = [];
      request.headers.forEach((value, key) => {
        headers.push([key, value]);
      });
      let meta;
      try {
        meta = fetchOpen({
          url: request.url,
          method: request.method,
          headers,
          body,
          redirect: request.redirect,
          credentials: request.credentials
        });
      } catch (error) {
        throw error instanceof Error ? error : new TypeError(String(error));
      }
      const stream = new ReadableStream({
        pull(controller) {
          let chunk;
          try {
            chunk = fetchRead(meta.id, 64 * 1024);
          } catch (error) {
            fetchClose(meta.id);
            controller.error(error instanceof Error ? error : new TypeError(String(error)));
            return;
          }
          if (chunk == null) {
            fetchClose(meta.id);
            controller.close();
            return;
          }
          const bytes = chunk instanceof Uint8Array ? chunk : new Uint8Array(chunk);
          if (bytes.byteLength === 0) {
            fetchClose(meta.id);
            controller.close();
            return;
          }
          controller.enqueue(bytes);
        },
        cancel(reason) {
          fetchAbort(meta.id, reason == null ? "" : String(reason));
        }
      });
      const response = new Response(null, {
        status: meta.status,
        statusText: meta.statusText,
        headers: meta.headers,
        url: meta.url
      });
      return attachResponseStreaming(response, stream);
    };
    globalThis.fetch.__gopeedOriginalFetch = originalFetch;
  }
})();
