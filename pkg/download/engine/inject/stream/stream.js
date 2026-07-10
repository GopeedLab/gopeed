(function () {
  const createBlobObjectURL = globalThis.__gopeed_create_blob_object_url;
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
      const contentType = this.headers && this.headers.get ? (this.headers.get("content-type") || "") : "";
      // Response.blob() is a materializing body consumer. Keeping the live
      // response stream behind an empty one-shot Blob makes retry, range and a
      // second object-URL reader silently produce an empty file.
      const bytes = await readAllFromStream(stream, false);
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

  const blobReaders = new Map();

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
    if (value && typeof value.getReader === "function") {
      return value.getReader();
    }
    throw new TypeError(sourceLabel + " must return a ReadableStream, got " + getValueTypeName(value));
  }

  async function openBlobReadable(blob, request) {
    const { offset, end } = request;
    const sliced = blob.slice(offset, end >= offset ? end + 1 : undefined);
    if (sliced && typeof sliced.stream === "function") {
      try {
        const stream = sliced.stream();
        if (stream && typeof stream.getReader === "function") {
          return stream;
        }
      } catch (_) {
      }
    }
    if (sliced && typeof sliced.arrayBuffer === "function") {
      const buffer = await sliced.arrayBuffer();
      return new ReadableStream({
        start(controller) {
          controller.enqueue(new Uint8Array(buffer));
          controller.close();
        },
      });
    }
    throw new TypeError("Blob source cannot be converted to a ReadableStream");
  }

  function describeObjectURLValue(value) {
    if (value instanceof Blob) {
      return {
        kind: "blob",
        value,
      };
    }
    if (typeof value === "function") {
      return {
        kind: "opener",
        openReadable: value,
        sourceLabel: "gopeed.runtime.blob.createObjectURL opener function",
      };
    }
    return {
      kind: "other",
      value,
    };
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

  function normalizeBlobObjectURLOptions(value, options, defaultRange) {
    const normalized = {
      contentType: "",
      size: 0,
      range: !!defaultRange,
    };
    if (options && typeof options === "object") {
      if (typeof options.contentType === "string") {
        normalized.contentType = options.contentType;
      }
      if (Number.isFinite(Number(options.size)) && Number(options.size) > 0) {
        normalized.size = Number(options.size);
      }
      if (typeof options.range === "boolean") {
        normalized.range = options.range;
      }
    }
    if (!normalized.contentType && value && typeof value.type === "string") {
      normalized.contentType = value.type;
    }
    if (!normalized.size && value) {
      if (Number.isFinite(Number(value.size)) && Number(value.size) > 0) {
        normalized.size = Number(value.size);
      } else if (value._buffer && Number.isFinite(Number(value._buffer.byteLength))) {
        normalized.size = Number(value._buffer.byteLength);
      }
    }
    return normalized;
  }

  function validateBlobObjectURLOptions(options) {
    if (options.range && !(Number.isFinite(Number(options.size)) && Number(options.size) > 0)) {
      throw new TypeError("gopeed.runtime.blob.createObjectURL options.range requires a positive options.size");
    }
  }

  function normalizeOpenRequest(request) {
    const offset = Math.max(0, Number(request && request.offset) || 0);
    const endValue = Number(request && request.end);
    const end = Number.isFinite(endValue) ? endValue : -1;
    return { offset, end };
  }

  function yieldBlobPipeTask() {
    return new Promise((resolve) => {
      if (typeof setTimeout === "function") {
        setTimeout(resolve, 0);
      } else {
        Promise.resolve().then(resolve);
      }
    });
  }

  function createPendingBlobReader(openReadable, request) {
    let activeReader;
    let ready;
    function ensureReader() {
      if (ready) {
        return ready;
      }
      ready = (async function () {
        const source = await openReadable(normalizeOpenRequest(request));
        activeReader = toReadableStreamReader(source, "gopeed.runtime.blob.createObjectURL opener function");
        return activeReader;
      })();
      return ready;
    }
    return {
      async read() {
        const reader = await ensureReader();
        return reader.read();
      },
      cancel(reason) {
        return ensureReader().then(function (reader) {
          return reader.cancel(reason);
        }, function () {});
      },
      releaseLock() {
        if (!ready) {
          return;
        }
        return ready.then(function (reader) {
          return reader.releaseLock();
        }, function () {});
      }
    };
  }

  globalThis.__gopeed_blob_open_source = function (openReadable, request) {
    if (typeof openReadable !== "function") {
      throw new TypeError("blob opener must be callable");
    }
    const id = crypto.randomUUID ? crypto.randomUUID() : String(Date.now()) + "-" + String(Math.random());
    blobReaders.set(id, createPendingBlobReader(openReadable, request));
    return id;
  };

  globalThis.__gopeed_blob_read_source = async function (id, chunkSize) {
    const reader = blobReaders.get(id);
    if (!reader) {
      return null;
    }
    const { done, value } = await reader.read();
    if (done) {
      blobReaders.delete(id);
      releaseReader(reader);
      return null;
    }
    const chunk = value instanceof Uint8Array ? value : new Uint8Array(value);
    const size = Math.max(1, Number(chunkSize) || chunk.byteLength);
    if (chunk.byteLength <= size) {
      return chunk.buffer.slice(chunk.byteOffset, chunk.byteOffset + chunk.byteLength);
    }
    const head = chunk.slice(0, size);
    let offset = size;
    blobReaders.set(id, {
      read() {
        if (offset >= chunk.byteLength) {
          return reader.read();
        }
        const next = chunk.slice(offset, Math.min(offset + size, chunk.byteLength));
        offset += next.byteLength;
        return Promise.resolve({ done: false, value: next });
      },
      cancel(reason) {
        return reader.cancel(reason);
      },
      releaseLock() {
        return reader.releaseLock();
      }
    });
    return head.buffer.slice(head.byteOffset, head.byteOffset + head.byteLength);
  };

  globalThis.__gopeed_blob_close_source = function (id) {
    const reader = blobReaders.get(id);
    if (!reader) {
      return;
    }
    blobReaders.delete(id);
    releaseReader(reader, "blob request closed");
  };

  const blobPipeReaderStates = new Map();

  function releaseBlobPipeReader(state) {
    if (!state || state.released) {
      return;
    }
    state.released = true;
    if (state.reader && typeof state.reader.releaseLock === "function") {
      try {
        state.reader.releaseLock();
      } catch (_) {
      }
    }
    if (blobPipeReaderStates.get(state.pipeId) === state) {
      blobPipeReaderStates.delete(state.pipeId);
    }
  }

  function finishBlobPipeReader(state, cancel, reason) {
    if (!state) {
      return Promise.resolve();
    }
    if (cancel) {
      state.cancelRequested = true;
      if (reason !== undefined) {
        state.cancelReason = reason;
      }
    }
    if (state.cleanupPromise) {
      return state.cleanupPromise;
    }
    if (!state.reader) {
      // A cancellation may arrive while the asynchronous opener is pending.
      // Keep the state until the reader exists so it can still be canceled.
      if (!state.cancelRequested || state.openSettled) {
        releaseBlobPipeReader(state);
      }
      return Promise.resolve();
    }

    state.cleanupPromise = (async function () {
      try {
        if (state.cancelRequested && typeof state.reader.cancel === "function") {
          await state.reader.cancel(state.cancelReason);
        }
      } catch (_) {
      } finally {
        releaseBlobPipeReader(state);
      }
    })();
    return state.cleanupPromise;
  }

  globalThis.__gopeed_blob_cancel_pipe_source = function (pipeId, reason) {
    const state = blobPipeReaderStates.get(pipeId);
    if (!state) {
      return;
    }
    // finishBlobPipeReader invokes reader.cancel() synchronously up to its
    // first await. Do not return the cleanup Promise: Go only needs to ensure
    // cancellation has started before releasing the source session.
    finishBlobPipeReader(state, true, reason).catch(function () {});
  };

  globalThis.__gopeed_blob_pipe_source = function (openReadable, request, pipeId) {
    if (typeof openReadable !== "function") {
      throw new TypeError("blob opener must be callable");
    }
    const state = {
      pipeId,
      reader: null,
      cancelRequested: false,
      cancelReason: "blob pipe closed",
      cleanupPromise: null,
      openSettled: false,
      released: false,
    };
    blobPipeReaderStates.set(pipeId, state);
    const startPipe = async function () {
      try {
        const source = await openReadable(normalizeOpenRequest(request));
        state.openSettled = true;
        state.reader = toReadableStreamReader(source, "gopeed.runtime.blob.createObjectURL opener function");
        if (state.cancelRequested) {
          await finishBlobPipeReader(state, true, state.cancelReason);
          return;
        }
        while (true) {
          const { done, value } = await state.reader.read();
          if (done) {
            globalThis.__gopeed_blob_pipe_close(pipeId);
            return;
          }
          const chunk = value instanceof Uint8Array ? value : new Uint8Array(value);
          const buffer = chunk.buffer.slice(chunk.byteOffset, chunk.byteOffset + chunk.byteLength);
          if (!globalThis.__gopeed_blob_pipe_chunk(pipeId, buffer)) {
            state.cancelRequested = true;
            state.cancelReason = "blob pipe closed";
            return;
          }
          await yieldBlobPipeTask();
        }
      } catch (error) {
        if (!state.cancelRequested) {
          globalThis.__gopeed_blob_pipe_error(pipeId, error && error.stack ? error.stack : String(error));
        }
      } finally {
        state.openSettled = true;
        await finishBlobPipeReader(state, state.cancelRequested, state.cancelReason);
      }
    };
    if (typeof setTimeout === "function") {
      setTimeout(startPipe, 0);
    } else {
      Promise.resolve().then(startPipe);
    }
  };

  globalThis.__gopeed_blob_create_object_url = function (value, options) {
    const described = describeObjectURLValue(value);
    if (described.kind === "blob") {
      const blob = described.value;
      const normalized = normalizeBlobObjectURLOptions(blob, options, Number(blob.size) > 0);
      validateBlobObjectURLOptions(normalized);
      const opener = async (request) => openBlobReadable(blob, request);
      return createBlobObjectURL(opener, normalized);
    }
    if (described.kind === "opener") {
      const normalized = normalizeBlobObjectURLOptions(null, options, false);
      validateBlobObjectURLOptions(normalized);
      return createBlobObjectURL(described.openReadable, normalized);
    }
    throw new TypeError("Unsupported object type for gopeed.runtime.blob.createObjectURL: " + getValueTypeName(described.value) + ". Expected Blob or opener function");
  };

  globalThis.__gopeed_blob_revoke_object_url = function (url) {
    if (typeof url === "string") {
      revokeObjectURL(url);
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
        meta = await fetchOpen({
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
        async pull(controller) {
          let chunk;
          try {
            chunk = await fetchRead(meta.id, 64 * 1024);
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
