(function () {
  const createBlobObjectURL = globalThis.__gopeed_create_blob_object_url;
  const createWritableObjectURL = globalThis.__gopeed_create_writable_stream_object_url;
  const writeWritableObjectURL = globalThis.__gopeed_write_writable_stream_object_url;
  const closeWritableObjectURL = globalThis.__gopeed_close_writable_stream_object_url;
  const abortWritableObjectURL = globalThis.__gopeed_abort_writable_stream_object_url;
  const revokeObjectURL = globalThis.__gopeed_revoke_object_url;

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
        this._queue = [];
        this._waiters = [];
        this._closed = false;
        this._errored = null;
        this._reader = null;
        this.locked = false;
        const controller = new ReadableStreamDefaultController(this);
        if (typeof source.start === "function") {
          source.start(controller);
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
        if (this._queue.length > 0) {
          return Promise.resolve({ done: false, value: this._queue.shift() });
        }
        if (this._closed) {
          return Promise.resolve({ done: true, value: undefined });
        }
        return new Promise((resolve, reject) => {
          this._waiters.push({ resolve, reject });
        });
      }

      getReader() {
        if (this.locked) {
          throw new TypeError("ReadableStream is locked");
        }
        this.locked = true;
        this._reader = new ReadableStreamDefaultReader(this);
        return this._reader;
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

  const originalCreateObjectURL = typeof URL.createObjectURL === "function"
    ? URL.createObjectURL.bind(URL)
    : null;
  const originalRevokeObjectURL = typeof URL.revokeObjectURL === "function"
    ? URL.revokeObjectURL.bind(URL)
    : null;

  URL.createObjectURL = function (value) {
    if (value instanceof Blob) {
      return createBlobObjectURL(value._buffer, value.type || "");
    }
    if (value instanceof WritableStream) {
      if (value.__gopeedObjectURL) {
        return value.__gopeedObjectURL;
      }
      const url = createWritableObjectURL();
      value._addObserver({
        write(chunk) {
          return Promise.resolve(writeWritableObjectURL(url, chunk));
        },
        close() {
          return Promise.resolve(closeWritableObjectURL(url));
        },
        abort(reason) {
          return Promise.resolve(abortWritableObjectURL(url, reason == null ? "" : String(reason)));
        }
      });
      value.__gopeedObjectURL = url;
      return url;
    }
    if (originalCreateObjectURL) {
      return originalCreateObjectURL(value);
    }
    throw new TypeError("Unsupported object type");
  };

  URL.revokeObjectURL = function (url) {
    if (typeof url === "string" && url.indexOf("gblob:") === 0) {
      revokeObjectURL(url);
      return;
    }
    if (originalRevokeObjectURL) {
      originalRevokeObjectURL(url);
    }
  };
})();
