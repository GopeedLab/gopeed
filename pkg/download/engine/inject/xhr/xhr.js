(function () {
  if (typeof globalThis.ProgressEvent === "undefined") {
    globalThis.ProgressEvent = class ProgressEvent extends Event {
      constructor(type, init) {
        super(type);
        init = init || {};
        this.lengthComputable = Boolean(init.lengthComputable);
        this.loaded = Number(init.loaded) || 0;
        this.total = Number(init.total) || 0;
      }
    };
  }

  function createProgressEvent(type, loaded, total) {
    const lengthComputable = Number.isFinite(total) && total >= 0;
    return new ProgressEvent(type, {
      lengthComputable,
      loaded: loaded || 0,
      total: lengthComputable ? total : 0,
    });
  }

  function canonicalHeaderName(name) {
    return String(name).toLowerCase().split("-").map(function (part) {
      return part ? part.charAt(0).toUpperCase() + part.slice(1) : part;
    }).join("-");
  }

  function bodySize(body) {
    if (body == null) return 0;
    if (typeof body === "string") return new TextEncoder().encode(body).byteLength;
    if (body instanceof ArrayBuffer) return body.byteLength;
    if (typeof ArrayBuffer !== "undefined" && ArrayBuffer.isView && ArrayBuffer.isView(body)) {
      return body.byteLength;
    }
    if (typeof Blob !== "undefined" && body instanceof Blob) return Number(body.size) || 0;
    if (body && Number.isFinite(Number(body.size))) return Number(body.size);
    return 0;
  }

  class XMLHttpRequestUpload extends EventTarget {
    constructor() {
      super();
      this.onloadstart = null;
      this.onprogress = null;
      this.onabort = null;
      this.onerror = null;
      this.onload = null;
      this.ontimeout = null;
      this.onloadend = null;
    }

    _emit(type, loaded, total) {
      const event = createProgressEvent(type, loaded, total);
      const handler = this["on" + type];
      if (typeof handler === "function") handler.call(this, event);
      this.dispatchEvent(event);
    }
  }

  class XMLHttpRequest extends EventTarget {
    constructor() {
      super();
      this.readyState = XMLHttpRequest.UNSENT;
      this.response = "";
      this.responseText = "";
      this.responseType = "";
      this.responseURL = "";
      this.responseXML = null;
      this.status = 0;
      this.statusText = "";
      this.timeout = 0;
      this.withCredentials = false;
      this.upload = new XMLHttpRequestUpload();
      this.redirect = "follow";
      this.onreadystatechange = null;
      this.onloadstart = null;
      this.onprogress = null;
      this.onabort = null;
      this.onerror = null;
      this.onload = null;
      this.ontimeout = null;
      this.onloadend = null;
      this._method = "GET";
      this._url = "";
      this._requestHeaders = new Headers();
      this._responseHeaders = null;
      this._controller = null;
      this._sendStarted = false;
      this._aborted = false;
      this._timedOut = false;
      this._timeoutID = null;
    }

    open(method, url, async) {
      if (arguments.length < 2) {
        throw new TypeError("XMLHttpRequest.open requires at least 2 arguments");
      }
      if (async === false) {
        throw new DOMException("Synchronous XMLHttpRequest is not supported", "NotSupportedError");
      }
      this._method = String(method).toUpperCase();
      this._url = String(url);
      this._requestHeaders = new Headers();
      this._responseHeaders = null;
      this._sendStarted = false;
      this._aborted = false;
      this._timedOut = false;
      this.response = "";
      this.responseText = "";
      this.responseURL = "";
      this.status = 0;
      this.statusText = "";
      this._setReadyState(XMLHttpRequest.OPENED);
    }

    setRequestHeader(name, value) {
      if (this.readyState !== XMLHttpRequest.OPENED || this._sendStarted) {
        throw new DOMException("XMLHttpRequest is not opened", "InvalidStateError");
      }
      this._requestHeaders.append(name, value);
    }

    getResponseHeader(name) {
      if (!this._responseHeaders || this.readyState < XMLHttpRequest.HEADERS_RECEIVED) return null;
      return this._responseHeaders.get(name);
    }

    getAllResponseHeaders() {
      if (!this._responseHeaders || this.readyState < XMLHttpRequest.HEADERS_RECEIVED) return "";
      let result = "";
      this._responseHeaders.forEach(function (value, name) {
        result += canonicalHeaderName(name) + ": " + value + "\r\n";
      });
      return result;
    }

    overrideMimeType() {
      // Kept as a compatibility no-op until MIME decoding is configurable.
    }

    send(body) {
      if (this.readyState !== XMLHttpRequest.OPENED || this._sendStarted) {
        throw new DOMException("XMLHttpRequest is not opened", "InvalidStateError");
      }
      this._sendStarted = true;
      this._aborted = false;
      this._timedOut = false;
      this._controller = new AbortController();
      const uploadTotal = bodySize(body);
      this._emit("loadstart", 0, -1);
      this.upload._emit("loadstart", 0, uploadTotal);

      if (this.timeout > 0) {
        this._timeoutID = setTimeout(() => {
          this._timedOut = true;
          this._controller.abort();
        }, this.timeout);
      }

      const options = {
        method: this._method,
        headers: this._requestHeaders,
        redirect: this.redirect || "follow",
        credentials: this.withCredentials ? "include" : "same-origin",
        signal: this._controller.signal,
      };
      if (this._method !== "GET" && this._method !== "HEAD" && body != null) {
        options.body = body;
      }

      Promise.resolve().then(() => globalThis.fetch(this._url, options)).then(async (response) => {
        if (this._aborted || this._timedOut) return;
        this.upload._emit("progress", uploadTotal, uploadTotal);
        this.upload._emit("load", uploadTotal, uploadTotal);
        this.upload._emit("loadend", uploadTotal, uploadTotal);

        this.status = response.status;
        this.statusText = response.statusText;
        this.responseURL = response.url || this._url;
        this._responseHeaders = response.headers;
        this._setReadyState(XMLHttpRequest.HEADERS_RECEIVED);

        const contentLengthValue = response.headers.get("content-length");
        const contentLength = contentLengthValue == null ? -1 : Number(contentLengthValue);
        const chunks = [];
        let loaded = 0;
        if (response.body) {
          const reader = response.body.getReader();
          try {
            while (true) {
              const item = await reader.read();
              if (item.done) break;
              const chunk = item.value instanceof Uint8Array ? item.value : new Uint8Array(item.value);
              chunks.push(chunk);
              loaded += chunk.byteLength;
              this._setReadyState(XMLHttpRequest.LOADING);
              this._emit("progress", loaded, contentLength);
            }
          } finally {
            if (typeof reader.releaseLock === "function") reader.releaseLock();
          }
        }

        const bytes = new Uint8Array(loaded);
        let offset = 0;
        for (const chunk of chunks) {
          bytes.set(chunk, offset);
          offset += chunk.byteLength;
        }
        const text = new TextDecoder().decode(bytes);
        switch (this.responseType) {
          case "arraybuffer":
            this.response = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength);
            break;
          case "blob":
            this.response = new Blob([bytes], { type: response.headers.get("content-type") || "" });
            break;
          case "json":
            try {
              this.response = text === "" ? null : JSON.parse(text);
            } catch (_) {
              this.response = null;
            }
            break;
          case "":
          case "text":
          default:
            this.response = text;
            this.responseText = text;
            break;
        }
        this._clearTimeout();
        this._setReadyState(XMLHttpRequest.DONE);
        this._emit("load", loaded, contentLength);
        this._emit("loadend", loaded, contentLength);
      }).catch((error) => {
        this._clearTimeout();
        if (this._timedOut) {
          this.status = 0;
          this._setReadyState(XMLHttpRequest.DONE);
          this.upload._emit("timeout", 0, uploadTotal);
          this._emit("timeout", 0, -1);
          this._emit("loadend", 0, -1);
          return;
        }
        if (this._aborted || (error && error.name === "AbortError")) {
          this.status = 0;
          this._setReadyState(XMLHttpRequest.DONE);
          this.upload._emit("abort", 0, uploadTotal);
          this._emit("abort", 0, -1);
          this._emit("loadend", 0, -1);
          return;
        }
        this.status = 0;
        this._setReadyState(XMLHttpRequest.DONE);
        this.upload._emit("error", 0, uploadTotal);
        this._emit("error", 0, -1);
        this._emit("loadend", 0, -1);
      });
    }

    abort() {
      if (!this._sendStarted || this.readyState === XMLHttpRequest.DONE) return;
      this._aborted = true;
      this._clearTimeout();
      if (this._controller) this._controller.abort();
    }

    _clearTimeout() {
      if (this._timeoutID != null) {
        clearTimeout(this._timeoutID);
        this._timeoutID = null;
      }
    }

    _setReadyState(state) {
      this.readyState = state;
      this._emit("readystatechange", 0, -1);
    }

    _emit(type, loaded, total) {
      const event = createProgressEvent(type, loaded, total);
      const handler = this["on" + type];
      if (typeof handler === "function") handler.call(this, event);
      this.dispatchEvent(event);
    }
  }

  XMLHttpRequest.UNSENT = 0;
  XMLHttpRequest.OPENED = 1;
  XMLHttpRequest.HEADERS_RECEIVED = 2;
  XMLHttpRequest.LOADING = 3;
  XMLHttpRequest.DONE = 4;
  XMLHttpRequest.prototype.UNSENT = 0;
  XMLHttpRequest.prototype.OPENED = 1;
  XMLHttpRequest.prototype.HEADERS_RECEIVED = 2;
  XMLHttpRequest.prototype.LOADING = 3;
  XMLHttpRequest.prototype.DONE = 4;

  globalThis.XMLHttpRequest = XMLHttpRequest;
})();
