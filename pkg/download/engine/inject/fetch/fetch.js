(function () {
  const fetchOpen = globalThis.__gopeed_fetch_open;
  const fetchRead = globalThis.__gopeed_fetch_read;
  const fetchClose = globalThis.__gopeed_fetch_close;
  const fetchAbort = globalThis.__gopeed_fetch_abort;

  class FormData {
    constructor(form) {
      if (form !== undefined) throw new TypeError("FormData HTML form construction is not available in an extension worker");
      this._entries = [];
    }

    _normalizeValue(value) {
      if (typeof Blob !== "undefined" && value instanceof Blob) return value;
      if (value && typeof value === "object" && "name" in value && "size" in value) return value;
      return String(value);
    }

    append(name, value) {
      this._entries.push([String(name), this._normalizeValue(value)]);
    }

    delete(name) {
      name = String(name);
      this._entries = this._entries.filter((entry) => entry[0] !== name);
    }

    get(name) {
      name = String(name);
      const entry = this._entries.find((item) => item[0] === name);
      return entry ? entry[1] : null;
    }

    getAll(name) {
      name = String(name);
      return this._entries.filter((entry) => entry[0] === name).map((entry) => entry[1]);
    }

    has(name) {
      name = String(name);
      return this._entries.some((entry) => entry[0] === name);
    }

    set(name, value) {
      name = String(name);
      value = this._normalizeValue(value);
      const first = this._entries.findIndex((entry) => entry[0] === name);
      if (first < 0) {
        this._entries.push([name, value]);
        return;
      }
      this._entries[first] = [name, value];
      this._entries = this._entries.filter((entry, index) => index === first || entry[0] !== name);
    }

    _iterator(kind) {
      const data = this;
      let index = 0;
      return {
        next() {
          if (index >= data._entries.length) return { value: undefined, done: true };
          const entry = data._entries[index++];
          if (kind === "key") return { value: entry[0], done: false };
          if (kind === "value") return { value: entry[1], done: false };
          return { value: [entry[0], entry[1]], done: false };
        },
        [Symbol.iterator]() { return this; },
      };
    }

    entries() { return this._iterator("entry"); }
    keys() { return this._iterator("key"); }
    values() { return this._iterator("value"); }
    [Symbol.iterator]() { return this.entries(); }

    forEach(callback, thisArg) {
      for (const [name, value] of this) callback.call(thisArg, value, name, this);
    }
  }
  Object.defineProperty(FormData.prototype, Symbol.toStringTag, { configurable: true, value: "FormData" });
  globalThis.FormData = FormData;

  const headerNamePattern = /^[!#$%&'*+\-.^_`|~0-9A-Za-z]+$/;
  const headersIteratorBase = Object.getPrototypeOf(Object.getPrototypeOf([][Symbol.iterator]()));
  const headersIteratorPrototype = Object.create(headersIteratorBase);

  function normalizeHeaderName(name) {
	if (typeof name === "symbol") throw new TypeError("HTTP header names must be ByteStrings");
    name = String(name);
	for (let index = 0; index < name.length; index++) {
	  if (name.charCodeAt(index) > 0xFF) throw new TypeError("HTTP header names must be ByteStrings");
	}
    if (!headerNamePattern.test(name)) throw new TypeError("Invalid HTTP header name");
    return name.toLowerCase();
  }

  function normalizeHeaderValue(value) {
    value = String(value);
    for (let index = 0; index < value.length; index++) {
      if (value.charCodeAt(index) > 0xFF) throw new TypeError("HTTP header values must be ByteStrings");
    }
    value = value.replace(/^[\t\n\r ]+|[\t\n\r ]+$/g, "");
    if (/[\0\n\r]/.test(value)) throw new TypeError("Invalid HTTP header value");
    return value;
  }

  function normalizeStatusText(value) {
    value = String(value);
    for (let index = 0; index < value.length; index++) {
      const code = value.charCodeAt(index);
      if (code > 0xFF || (code !== 0x09 && (code < 0x20 || code === 0x7F))) {
        throw new TypeError("Invalid response status text");
      }
    }
    return value;
  }

  function sortedHeaderEntries(headers) {
    const result = [];
    for (const name of Array.from(headers._values.keys()).sort()) {
      const values = headers._values.get(name);
      if (name === "set-cookie") {
        for (const value of values) result.push([name, value]);
      } else {
        result.push([name, values.join(", ")]);
      }
    }
    return result;
  }

  Object.defineProperty(headersIteratorPrototype, "next", {
    configurable: true,
    enumerable: true,
    writable: true,
    value: function () {
      const entries = sortedHeaderEntries(this._headers);
      if (this._index >= entries.length) return { value: undefined, done: true };
      const entry = entries[this._index++];
      if (this._kind === "key") return { value: entry[0], done: false };
      if (this._kind === "value") return { value: entry[1], done: false };
      return { value: entry, done: false };
    },
  });
  Object.defineProperty(headersIteratorPrototype, Symbol.iterator, {
    configurable: true,
    enumerable: false,
    writable: true,
    value: function () { return this; },
  });

  function createHeadersIterator(headers, kind) {
    const iterator = Object.create(headersIteratorPrototype);
    Object.defineProperties(iterator, {
      _headers: { value: headers },
      _kind: { value: kind },
      _index: { value: 0, writable: true },
    });
    return iterator;
  }

  function forbiddenRequestHeader(name, value) {
    if (name.startsWith("proxy-") || name.startsWith("sec-")) return true;
    if ([
      "accept-charset", "accept-encoding", "access-control-request-headers",
      "access-control-request-method", "connection", "content-length", "cookie",
      "cookie2", "date", "dnt", "expect", "host", "keep-alive", "origin",
      "permissions-policy", "referer", "te", "trailer", "transfer-encoding",
      "set-cookie", "upgrade", "via",
    ].includes(name)) return true;
    if (["x-http-method", "x-http-method-override", "x-method-override"].includes(name)) {
      return String(value).split(",").some((method) => {
        method = method.trim().toUpperCase();
        return method === "CONNECT" || method === "TRACE" || method === "TRACK";
      });
    }
    return false;
  }

  function noCORSSafelistedRequestHeader(name, value) {
    if (value.length > 128) return false;
    const hasUnsafeByte = /[\x00-\x08\x0A-\x1F\x7F"():<>?@\[\\\]{}]/.test(value);
    if (name === "accept") return !hasUnsafeByte;
    if (name === "accept-language" || name === "content-language") {
      return /^[0-9A-Za-z *,\-.;=]*$/.test(value);
    }
    if (name === "content-type") {
      if (hasUnsafeByte) return false;
      const essence = value.split(";", 1)[0].trim().toLowerCase();
      return essence === "application/x-www-form-urlencoded" ||
        essence === "multipart/form-data" || essence === "text/plain";
    }
    if (name === "range") return /^bytes=[0-9]+-[0-9]*$/.test(value);
    return false;
  }

  function blockedByGuard(headers, name, value) {
    if (headers._guard === "immutable") throw new TypeError("Headers are immutable");
    if (headers._guard === "request") return forbiddenRequestHeader(name, value);
    if (headers._guard === "request-no-cors") {
      return forbiddenRequestHeader(name, value) || !noCORSSafelistedRequestHeader(name, value);
    }
    if (headers._guard === "response") return name === "set-cookie" || name === "set-cookie2";
    return false;
  }

  class Headers {
    constructor(init, guard) {
      this._values = new Map();
	  this._guard = guard || "none";
      if (init === undefined) return;
      if (init === null || (typeof init !== "object" && typeof init !== "function")) {
        throw new TypeError("Headers init must be a sequence or record");
      }
      const iterator = init[Symbol.iterator];
      if (typeof iterator === "function") {
        for (const pair of init) {
          if (pair == null || typeof pair[Symbol.iterator] !== "function") {
            throw new TypeError("Header pair must be iterable");
          }
          const values = Array.from(pair);
          if (values.length !== 2) throw new TypeError("Header pair must contain exactly two items");
          this.append(values[0], values[1]);
        }
        return;
      }
	  for (const name of Reflect.ownKeys(init)) {
		const descriptor = Reflect.getOwnPropertyDescriptor(init, name);
		if (!descriptor || !descriptor.enumerable) continue;
		const normalizedName = normalizeHeaderName(name);
		this.append(normalizedName, init[name]);
	  }
    }

    append(name, value) {
      name = normalizeHeaderName(name);
      value = normalizeHeaderValue(value);
      const values = this._values.get(name);
      const combinedValue = values && name !== "set-cookie" ? values.join(", ") + ", " + value : value;
	  if (blockedByGuard(this, name, combinedValue)) return;
      if (values) values.push(value);
      else this._values.set(name, [value]);
    }

    delete(name) {
	  name = normalizeHeaderName(name);
	  if (this._guard === "immutable") throw new TypeError("Headers are immutable");
	  if (this._guard === "request-no-cors") {
		if (forbiddenRequestHeader(name, "") || !["accept", "accept-language", "content-language", "content-type", "range"].includes(name)) return;
		this._values.delete(name);
		return;
	  }
	  if (blockedByGuard(this, name, "")) return;
      this._values.delete(name);
    }

    get(name) {
      const values = this._values.get(normalizeHeaderName(name));
      return values ? values.join(", ") : null;
    }

    getSetCookie() {
      const values = this._values.get("set-cookie");
      return values ? values.slice() : [];
    }

    has(name) {
      return this._values.has(normalizeHeaderName(name));
    }

    set(name, value) {
	  name = normalizeHeaderName(name);
	  value = normalizeHeaderValue(value);
	  if (blockedByGuard(this, name, value)) return;
	  this._values.set(name, [value]);
    }

    keys() { return createHeadersIterator(this, "key"); }
    values() { return createHeadersIterator(this, "value"); }
    entries() { return createHeadersIterator(this, "entry"); }
    [Symbol.iterator]() { return this.entries(); }

    forEach(callback, thisArg) {
      for (const [name, value] of this) callback.call(thisArg, value, name, this);
    }
  }

  globalThis.Headers = Headers;

  const forbiddenMethods = new Set(["CONNECT", "TRACE", "TRACK"]);

  function normalizeMethod(method) {
    method = String(method);
    if (!headerNamePattern.test(method)) throw new TypeError("Invalid HTTP method");
    method = method.toUpperCase();
    if (forbiddenMethods.has(method)) throw new TypeError("Forbidden HTTP method");
    return method;
  }

  function resolveURL(value) {
    const base = globalThis.location && globalThis.location.href ? globalThis.location.href : "http://localhost/";
    const parsed = new URL(String(value), base);
    if (parsed.username || parsed.password) throw new TypeError("Request URLs cannot contain credentials");
    return parsed.href;
  }

  function normalizeEnum(value, allowed, name) {
    value = String(value);
    if (!allowed.includes(value)) throw new TypeError("Invalid Request " + name);
    return value;
  }

  function normalizeReferrer(value) {
    value = String(value);
    if (value === "") return "no-referrer";
    if (value === "about:client") return value;
    return resolveURL(value);
  }

  function copyArrayBuffer(value) {
    if (value instanceof ArrayBuffer) return value.slice(0);
    return value.buffer.slice(value.byteOffset, value.byteOffset + value.byteLength);
  }

  function initializeBody(owner, body) {
    owner._bodyInit = body == null ? null : body;
    owner._bodyText = null;
    owner._bodyArrayBuffer = null;
    owner._bodyBlob = null;
    owner._bodyFormData = null;
    owner._bodyFormDataBoundary = "";
    owner._bodyStream = null;
    owner._bodyUsed = false;
    if (body == null) return;
    if (typeof ReadableStream !== "undefined" && body instanceof ReadableStream) {
      owner._bodyStream = body;
      body.__gopeedMarkBodyUsed = () => { owner._bodyUsed = true; };
    } else if (typeof body === "string") {
      owner._bodyText = body;
    } else if (typeof Blob !== "undefined" && body instanceof Blob) {
      owner._bodyBlob = body;
    } else if (typeof FormData !== "undefined" && body instanceof FormData) {
      owner._bodyFormData = body;
      const suffix = crypto.randomUUID ? crypto.randomUUID() : String(Date.now()) + String(Math.random()).slice(2);
      owner._bodyFormDataBoundary = "----gopeed-" + suffix;
    } else if (body instanceof ArrayBuffer || (ArrayBuffer.isView && ArrayBuffer.isView(body))) {
      owner._bodyArrayBuffer = copyArrayBuffer(body);
    } else if (typeof URLSearchParams !== "undefined" && body instanceof URLSearchParams) {
      owner._bodyText = body.toString();
    } else {
      owner._bodyText = String(body);
    }
  }

  function bodyContentType(owner) {
    if (owner._bodyText != null) {
      if (typeof URLSearchParams !== "undefined" && owner._bodyInit instanceof URLSearchParams) {
        return "application/x-www-form-urlencoded;charset=UTF-8";
      }
      return "text/plain;charset=UTF-8";
    }
    if (owner._bodyBlob && owner._bodyBlob.type) return owner._bodyBlob.type;
    if (owner._bodyFormData) return "multipart/form-data; boundary=" + owner._bodyFormDataBoundary;
    return "";
  }

  function decodeFormComponent(value) {
    return decodeURIComponent(String(value).replace(/\+/g, " "));
  }

  function installBodyMethods(prototype) {
    Object.defineProperties(prototype, {
      body: {
        configurable: true,
        enumerable: true,
        get() {
          if (this._bodyInit == null && !this.__gopeedBodyStream) return null;
          if (this.__gopeedBodyStream) return this.__gopeedBodyStream;
          if (!this._bodyStream) this._bodyStream = createBodyReadableStream(this);
          return this._bodyStream;
        },
      },
      bodyUsed: {
        configurable: true,
        enumerable: true,
        get() { return Boolean(this._bodyUsed || this.__gopeedBodyConsumed); },
      },
    });
    prototype.text = async function () {
      if (this.bodyUsed) throw new TypeError("Already read");
      if (this.body == null) return "";
      return readAllFromStream(this.body, true);
    };
    prototype.arrayBuffer = async function () {
      if (this.bodyUsed) throw new TypeError("Already read");
      if (this.body == null) return new ArrayBuffer(0);
      const bytes = await readAllFromStream(this.body, false);
      return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength);
    };
    prototype.bytes = async function () {
      if (this.bodyUsed) throw new TypeError("Already read");
      if (this.body == null) return new Uint8Array(0);
      return readAllFromStream(this.body, false);
    };
    prototype.blob = async function () {
      const data = await this.arrayBuffer();
      return new Blob([data], { type: this.headers.get("content-type") || "" });
    };
    prototype.json = async function () { return JSON.parse(await this.text()); };
    prototype.formData = async function () {
      const contentType = this.headers.get("content-type") || "";
      const essence = contentType.split(";", 1)[0].trim().toLowerCase();
      if (essence === "multipart/form-data" && this._bodyFormData) {
        if (this.bodyUsed) throw new TypeError("Already read");
        await readAllFromStream(this.body, false);
        const result = new FormData();
        for (const entry of Array.from(this._bodyFormData.entries())) result.append(entry[0], entry[1]);
        return result;
      }
      if (essence !== "application/x-www-form-urlencoded") {
        throw new TypeError("Body is not URL-encoded form data");
      }
      if (this.body == null) return new FormData();
      const text = await this.text();
      const result = new FormData();
      for (const entry of text.split("&")) {
        if (!entry) continue;
        const parts = entry.split("=");
        const name = parts.shift();
        result.append(decodeFormComponent(name), decodeFormComponent(parts.join("=")));
      }
      return result;
    };
  }

  function transferReadableStream(stream) {
    let reader = null;
    return new ReadableStream({
      async pull(controller) {
        if (!reader) reader = stream.getReader();
        const result = await reader.read();
        if (result.done) {
          reader.releaseLock();
          controller.close();
        } else {
          controller.enqueue(result.value);
        }
      },
      cancel(reason) {
        if (!reader) reader = stream.getReader();
        return reader.cancel(reason);
      },
    });
  }

  class Request {
    constructor(input, init) {
      if (arguments.length < 1) throw new TypeError("Request requires an input");
      init = init || {};
      if (init.window !== undefined && init.window !== null) throw new TypeError("Request window must be null");
      const source = input instanceof Request ? input : null;
      const bodyWasOverridden = Object.prototype.hasOwnProperty.call(init, "body");
      if (source && source.bodyUsed && !bodyWasOverridden) throw new TypeError("Request body is already used");
      this._url = source ? source.url : resolveURL(input);
      this._method = normalizeMethod(init.method !== undefined ? init.method : (source ? source.method : "GET"));
      this._destination = "";
      this._referrer = init.referrer !== undefined ? normalizeReferrer(init.referrer) : (source ? source.referrer : "about:client");
      this._referrerPolicy = normalizeEnum(init.referrerPolicy !== undefined ? init.referrerPolicy : (source ? source.referrerPolicy : ""), ["", "no-referrer", "no-referrer-when-downgrade", "origin", "origin-when-cross-origin", "same-origin", "strict-origin", "strict-origin-when-cross-origin", "unsafe-url"], "referrerPolicy");
      this._mode = normalizeEnum(init.mode !== undefined ? init.mode : (source ? source.mode : "cors"), ["same-origin", "no-cors", "cors"], "mode");
      this._credentials = normalizeEnum(init.credentials !== undefined ? init.credentials : (source ? source.credentials : "same-origin"), ["omit", "same-origin", "include"], "credentials");
      this._cache = normalizeEnum(init.cache !== undefined ? init.cache : (source ? source.cache : "default"), ["default", "no-store", "reload", "no-cache", "force-cache", "only-if-cached"], "cache");
      this._redirect = normalizeEnum(init.redirect !== undefined ? init.redirect : (source ? source.redirect : "follow"), ["follow", "error", "manual"], "redirect");
      if (this._mode === "no-cors" && !["GET", "HEAD", "POST"].includes(this._method)) {
        throw new TypeError("no-cors requests require a CORS-safelisted method");
      }
      if (this._cache === "only-if-cached" && this._mode !== "same-origin") {
        throw new TypeError("only-if-cached requires same-origin mode");
      }
      this._integrity = init.integrity !== undefined ? String(init.integrity) : (source ? source.integrity : "");
      this._keepalive = init.keepalive !== undefined ? Boolean(init.keepalive) : (source ? source.keepalive : false);
      this._priority = normalizeEnum(init.priority !== undefined ? init.priority : (source ? source._priority : "auto"), ["high", "low", "auto"], "priority");
      this._signal = init.signal || (source && source.signal) || new AbortController().signal;
      if (init.duplex !== undefined && String(init.duplex) !== "half") {
        throw new TypeError("Request duplex must be 'half'");
      }
      this._duplex = "half";
      const headerInit = init.headers !== undefined ? init.headers : (source ? source.headers : undefined);
      this._headers = new Headers(headerInit, this._mode === "no-cors" ? "request-no-cors" : "request");
      let body = init.body !== undefined ? init.body : (source ? source._bodyInit : null);
      if (source && !bodyWasOverridden && source.body) body = transferReadableStream(source.body);
      const bodyIsStream = typeof ReadableStream !== "undefined" && body instanceof ReadableStream;
      if (bodyIsStream && (body.locked || body._disturbed)) {
        throw new TypeError("ReadableStream body is locked or disturbed");
      }
      if (bodyIsStream && init.body !== undefined && init.duplex === undefined) {
        throw new TypeError("ReadableStream request bodies require duplex: 'half'");
      }
      if (bodyIsStream && this._keepalive) {
        throw new TypeError("keepalive requests cannot have a ReadableStream body");
      }
      if (source && !bodyWasOverridden && source.body && (source.body.locked || source.body._disturbed)) {
        throw new TypeError("Request body is locked or disturbed");
      }
      if ((this._method === "GET" || this._method === "HEAD") && body != null) {
        throw new TypeError("Body not allowed for GET or HEAD requests");
      }
      initializeBody(this, body);
      const contentType = bodyContentType(this);
      if (contentType && !this._headers.has("content-type")) this._headers.set("content-type", contentType);
      if (source && source.body) source._bodyUsed = true;
    }

    get method() { return this._method; }
    get url() { return this._url; }
    get headers() { return this._headers; }
    get destination() { return this._destination; }
    get referrer() { return this._referrer; }
    get referrerPolicy() { return this._referrerPolicy; }
    get mode() { return this._mode; }
    get credentials() { return this._credentials; }
    get cache() { return this._cache; }
    get redirect() { return this._redirect; }
    get integrity() { return this._integrity; }
    get keepalive() { return this._keepalive; }
    get signal() { return this._signal; }
    get isReloadNavigation() { return false; }
    get isHistoryNavigation() { return false; }
    get duplex() { return this._duplex; }
    clone() {
      if (this.bodyUsed) throw new TypeError("Request body is already used");
      let body = null;
      if (this.body) {
        const branches = this.body.tee();
        this._bodyInit = branches[0];
        this._bodyStream = branches[0];
        if (this.__gopeedBodyStream) this.__gopeedBodyStream = branches[0];
        branches[0].__gopeedMarkBodyUsed = () => { this._bodyUsed = true; };
        body = branches[1];
      }
      return new Request(this.url, {
        method: this.method,
        headers: this.headers,
        body,
        mode: this.mode,
        credentials: this.credentials,
        cache: this.cache,
        redirect: this.redirect,
        referrer: this.referrer,
        referrerPolicy: this.referrerPolicy,
        integrity: this.integrity,
        keepalive: this.keepalive,
        signal: this.signal,
        duplex: "half",
      });
    }
  }
  installBodyMethods(Request.prototype);

  const nullBodyStatuses = new Set([101, 103, 204, 205, 304]);

  class Response {
    constructor(body, init) {
      init = init || {};
      const status = init.status === undefined ? 200 : Number(init.status);
      if (!Number.isInteger(status) || status < 200 || status > 599) {
        throw new RangeError("Invalid response status");
      }
      if (body != null && nullBodyStatuses.has(status)) {
        throw new TypeError("Response with null-body status cannot have a body");
      }
      const statusText = init.statusText === undefined ? "" : normalizeStatusText(init.statusText);
      this._type = "default";
      this._url = init.url ? String(init.url) : "";
      this._redirected = Boolean(init.redirected);
      this._status = status;
      this._statusText = statusText;
      this._headers = new Headers(init.headers, "response");
      if (typeof ReadableStream !== "undefined" && body instanceof ReadableStream && (body.locked || body._disturbed)) {
        throw new TypeError("ReadableStream body is locked or disturbed");
      }
      initializeBody(this, body == null ? null : body);
      const contentType = bodyContentType(this);
      if (contentType && !this._headers.has("content-type")) this._headers.set("content-type", contentType);
    }

    get type() { return this._type; }
    get url() { return this._url; }
    get redirected() { return this._redirected; }
    get status() { return this._status; }
    get ok() { return this._status >= 200 && this._status <= 299; }
    get statusText() { return this._statusText; }
    get headers() { return this._headers; }
    clone() {
      if (this.bodyUsed) throw new TypeError("Response body is already used");
      let body = null;
      if (this.body) {
        const branches = this.body.tee();
        this._bodyInit = branches[0];
        this._bodyStream = branches[0];
        if (this.__gopeedBodyStream) this.__gopeedBodyStream = branches[0];
        branches[0].__gopeedMarkBodyUsed = () => { this._bodyUsed = true; };
        body = branches[1];
      }
      return new Response(body, {
        status: this.status,
        statusText: this.statusText,
        headers: this.headers,
        url: this.url,
        redirected: this.redirected,
      });
    }

    static error() {
      const response = Object.create(Response.prototype);
      response._type = "error";
      response._url = "";
      response._redirected = false;
      response._status = 0;
      response._statusText = "";
      response._headers = new Headers(undefined, "immutable");
      initializeBody(response, null);
      return response;
    }

    static redirect(url, status) {
      status = status === undefined ? 302 : Number(status);
      if (![301, 302, 303, 307, 308].includes(status)) throw new RangeError("Invalid redirect status");
      return new Response(null, { status, headers: { location: resolveURL(url) } });
    }

    static json(data, init) {
      const body = JSON.stringify(data);
      if (body === undefined) throw new TypeError("Value is not JSON serializable");
      init = Object.assign({}, init || {});
      const headers = new Headers(init.headers);
      if (!headers.has("content-type")) headers.set("content-type", "application/json");
      init.headers = headers;
      return new Response(body, init);
    }
  }
  installBodyMethods(Response.prototype);

  globalThis.Request = Request;
  globalThis.Response = Response;

  function toUint8Array(chunk) {
    if (chunk == null) return new Uint8Array(0);
    if (chunk instanceof Uint8Array) return chunk;
    if (typeof chunk === "string") return new TextEncoder().encode(chunk);
    if (chunk instanceof ArrayBuffer) return new Uint8Array(chunk);
    if (typeof ArrayBuffer !== "undefined" && ArrayBuffer.isView && ArrayBuffer.isView(chunk)) {
      return new Uint8Array(chunk.buffer, chunk.byteOffset, chunk.byteLength);
    }
    if (typeof Blob !== "undefined" && chunk instanceof Blob && chunk._buffer instanceof Uint8Array) {
      return chunk._buffer;
    }
    return new Uint8Array(0);
  }

  function createBodyReadableStream(owner) {
    const stream = new ReadableStream({
      start(controller) {
        Promise.resolve().then(async () => {
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
          if (owner._bodyFormData) {
            const boundary = owner._bodyFormDataBoundary;
            const entries = Array.from(owner._bodyFormData.entries());
            if (entries.length === 0) {
              controller.close();
              return;
            }
            let serialized = "";
            for (const entry of entries) {
              const name = String(entry[0]).replace(/\r|\n|"/g, (character) =>
                character === "\r" ? "%0D" : character === "\n" ? "%0A" : "%22");
              serialized += "--" + boundary + "\r\nContent-Disposition: form-data; name=\"" + name + "\"\r\n\r\n";
              serialized += String(entry[1]) + "\r\n";
            }
            serialized += "--" + boundary + "--\r\n";
            controller.enqueue(toUint8Array(serialized));
            controller.close();
            return;
          }
          if (owner._bodyText != null) {
            controller.enqueue(toUint8Array(owner._bodyText));
            controller.close();
            return;
          }
          if (owner._bodyInit != null) controller.enqueue(toUint8Array(owner._bodyInit));
          controller.close();
        }).catch((error) => controller.error(error));
      },
    });
	stream.__gopeedMarkBodyUsed = () => { owner._bodyUsed = true; };
	return stream;
  }

  async function readAllFromStream(stream, asText) {
    if (!stream) return asText ? "" : new Uint8Array(0);
    const reader = stream.getReader();
    const chunks = [];
    let total = 0;
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        if (!(value instanceof Uint8Array)) {
          throw new TypeError("Fetch body stream chunks must be Uint8Array values");
        }
        const chunk = value;
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
    return asText ? new TextDecoder().decode(merged) : merged;
  }

  function attachResponseStreaming(response, stream) {
    response.__gopeedBodyStream = stream;
    response.__gopeedBodyConsumed = false;
    const ensureUnused = () => {
      if (response.__gopeedBodyConsumed) throw new TypeError("Already read");
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
      markBodyUsed();
      return readAllFromStream(this.body, true);
    };
    response.arrayBuffer = async function () {
      markBodyUsed();
      const bytes = await readAllFromStream(this.body, false);
      return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength);
    };
    response.bytes = async function () {
      markBodyUsed();
      return readAllFromStream(this.body, false);
    };
    response.blob = async function () {
      markBodyUsed();
      const contentType = this.headers && this.headers.get ? (this.headers.get("content-type") || "") : "";
      const bytes = await readAllFromStream(this.body, false);
      return new Blob([bytes], { type: contentType });
    };
    response.json = async function () {
      return JSON.parse(await this.text());
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
          if (!this.__gopeedBodyStream) this.__gopeedBodyStream = createBodyReadableStream(this);
          return this.__gopeedBodyStream;
        },
      });
    }
  }

  if (typeof fetchOpen !== "function") return;

  globalThis.fetch = async function (input, init) {
    const request = new Request(input, init);
    const requestID = crypto.randomUUID ? crypto.randomUUID() : String(Date.now()) + "-" + String(Math.random());
    const signal = request.signal;
    let abortHandler = null;
    const cleanupAbort = () => {
      if (signal && abortHandler && typeof signal.removeEventListener === "function") {
        signal.removeEventListener("abort", abortHandler);
      }
      abortHandler = null;
    };
    if (signal) {
      if (signal.aborted) throw new DOMException("Aborted", "AbortError");
      abortHandler = () => fetchAbort(requestID, signal.reason == null ? "" : String(signal.reason));
      signal.addEventListener("abort", abortHandler, { once: true });
    }

    let body = null;
    if (request._bodyStream && !request._bodyFormData) {
      const bytes = await readAllFromStream(request._bodyStream, false);
      body = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength);
    } else if (request._bodyFormData) {
      body = request._bodyFormData._entries;
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
    request.headers.forEach((value, key) => headers.push([key, value]));
    let metadata;
    try {
      metadata = await fetchOpen({
        id: requestID,
        url: request.url,
        method: request.method,
        headers,
        body,
        bodyType: request._bodyFormData ? "formdata" : "",
        bodyBoundary: request._bodyFormDataBoundary,
        redirect: request.redirect,
        credentials: request.credentials,
      });
    } catch (error) {
      cleanupAbort();
      if (signal && signal.aborted) throw new DOMException("Aborted", "AbortError");
      throw error instanceof Error ? error : new TypeError(String(error));
    }

    const stream = new ReadableStream({
      async pull(controller) {
        let chunk;
        try {
          chunk = await fetchRead(metadata.id, 64 * 1024);
        } catch (error) {
          fetchClose(metadata.id);
          cleanupAbort();
          controller.error(error instanceof Error ? error : new TypeError(String(error)));
          return;
        }
        if (chunk == null) {
          fetchClose(metadata.id);
          cleanupAbort();
          controller.close();
          return;
        }
        const bytes = chunk instanceof Uint8Array ? chunk : new Uint8Array(chunk);
        if (bytes.byteLength === 0) {
          fetchClose(metadata.id);
          cleanupAbort();
          controller.close();
          return;
        }
        controller.enqueue(bytes);
      },
      cancel(reason) {
        cleanupAbort();
        fetchAbort(metadata.id, reason == null ? "" : String(reason));
      },
    });
    const response = new Response(null, {
      status: metadata.status,
      statusText: metadata.statusText,
      headers: metadata.headers,
      url: metadata.url,
      redirected: metadata.redirected,
    });
    response._type = "basic";
    response._headers._guard = "immutable";
    return attachResponseStreaming(response, stream);
  };
})();
