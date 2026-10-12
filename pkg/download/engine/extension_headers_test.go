package engine_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/download/engine"
)

// Exercise server-side header permissions, including formerly forbidden names,
// download authentication, metadata and non-safelisted no-cors values.
var extensionRequestHeaders = [][2]string{
	{"Accept-Charset", "utf-8"}, {"Accept-Encoding", "identity"},
	{"Access-Control-Request-Headers", "authorization, x-download"},
	{"Access-Control-Request-Method", "POST"}, {"Connection", "close"},
	{"Connection", "keep-alive"},
	{"Content-Length", "4"}, {"Cookie", "sid=manual"}, {"Cookie2", "legacy=1"},
	{"Date", "Fri, 09 Oct 2026 00:00:00 GMT"}, {"DNT", "1"},
	{"Host", "download.example:1234"}, {"Origin", "https://download.example"},
	{"Permissions-Policy", "camera=()"}, {"Referer", "https://download.example/file"},
	{"TE", "trailers"}, {"Trailer", "X-Checksum"}, {"Set-Cookie", "sid=request"},
	{"Set-Cookie2", "legacy=request"}, {"Via", "1.1 proxy"},
	{"Proxy-Authorization", "Basic dGVzdA=="}, {"Proxy-", "custom"},
	{"Sec-Fetch-Site", "same-origin"}, {"Sec-Fetch-Mode", "custom"},
	{"Sec-CH-UA", `"Download client"`}, {"Sec-", "custom"},
	{"X-HTTP-Method", "TRACE"}, {"X-HTTP-Method-Override", "CONNECT"},
	{"X-Method-Override", "TRACK"}, {"Authorization", "Bearer token"},
	{"User-Agent", "Gopeed extension"}, {"Content-Type", "application/json"},
	{"Range", "bytes=0-1,3-4"}, {"X-Download-Token", "custom"},
	{"Accept-Language", strings.Repeat("x", 129)},
}

func TestExtensionRequestsSendNodeHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		name := request.URL.Query().Get("header")
		value := request.Header.Get(name)
		if name == "Host" {
			value = request.Host
		}
		_ = json.NewEncoder(w).Encode([]string{value, string(body)})
	}))
	t.Cleanup(server.Close)
	clients := []struct{ name, script string }{
		{"fetch record", `const response = await fetch(url, {method: "POST", body: "test", headers: {[name]: value}}); return (await response.text()).trim();`},
		{"fetch no-cors Headers", `const response = await fetch(url, {method: "POST", body: "test", mode: "no-cors", headers: new Headers([[name.toLowerCase(), value]])}); return (await response.text()).trim();`},
		{"fetch Request clone", `const request = new Request(url, {method: "POST", body: "test", mode: "same-origin", headers: [[name.toUpperCase(), value]]}); const response = await fetch(request.clone()); return (await response.text()).trim();`},
	}
	for _, credentials := range []bool{false, true} {
		clients = append(clients, struct{ name, script string }{
			fmt.Sprintf("XHR credentials=%t", credentials),
			fmt.Sprintf(`return new Promise((resolve, reject) => {
				const xhr = new XMLHttpRequest();
				xhr.open("POST", url);
				xhr.withCredentials = %t;
				xhr.setRequestHeader(name.toLowerCase(), value);
				xhr.onload = () => resolve(xhr.responseText.trim());
				xhr.onerror = () => reject(new Error("XHR failed"));
				xhr.send("test");
			});`, credentials),
		})
	}
	for _, header := range extensionRequestHeaders {
		for _, client := range clients {
			t.Run(header[0]+"/"+client.name, func(t *testing.T) {
				runtime := engine.NewEngine(nil)
				t.Cleanup(runtime.Close)
				value, err := runtime.RunString(fmt.Sprintf(`(async () => {
					const name = %q, value = %q;
					const url = %q + "?header=" + encodeURIComponent(name);
					%s
				})()`, header[0], header[1], server.URL, client.script))
				if err != nil {
					t.Fatal(err)
				}
				want, _ := json.Marshal([]string{header[1], "test"})
				if value != string(want) {
					t.Fatalf("server received %#v, want %s", value, want)
				}
			})
		}
	}
}

func TestExtensionHeaderMutationsMatchNode(t *testing.T) {
	headers := append([][2]string(nil), extensionRequestHeaders...)
	// Node stores these in Headers/Request, but rejects them during fetch.
	headers = append(headers, [2]string{"Expect", "100-continue"}, [2]string{"Keep-Alive", "timeout=5"}, [2]string{"Transfer-Encoding", "chunked"}, [2]string{"Upgrade", "websocket"})
	data, _ := json.Marshal(headers)
	runtime := engine.NewEngine(nil)
	t.Cleanup(runtime.Close)
	_, err := runtime.RunString(fmt.Sprintf(`(() => {
		for (const mode of ["cors", "same-origin", "no-cors"]) {
			for (const [name, value] of %s) {
				const request = new Request("http://example.com", {mode, headers: {[name]: value}});
				const headers = request.headers;
				if (headers.get(name) !== value) throw new Error("Filtered init " + name + " in " + mode);
				headers.delete(name.toUpperCase());
				if (headers.has(name)) throw new Error("Cannot delete " + name);
				headers.append(name, value);
				if (headers.get(name) !== value) throw new Error("Cannot append " + name);
				headers.set(name, value);
				if (request.clone().headers.get(name) !== value) throw new Error("Cannot clone " + name);
			}
		}
		return true;
	})()`, data))
	if err != nil {
		t.Fatal(err)
	}
}

func TestExtensionRequestsIgnoreUnsupportedTransportHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		value := request.Header.Get(request.URL.Query().Get("header"))
		_ = json.NewEncoder(w).Encode([]string{value, string(body)})
	}))
	t.Cleanup(server.Close)
	for _, header := range [][2]string{
		{"Expect", "100-continue"}, {"Keep-Alive", "timeout=5"},
		{"Transfer-Encoding", "chunked"}, {"Upgrade", "websocket"},
		{"Connection", "upgrade"}, {"Content-Length", "3"},
		{"Content-Length", "5"}, {"Content-Length", "invalid"},
		{"Content-Length", "-1"}, {"Content-Length", ""},
	} {
		for _, xhr := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s=%s/XHR=%t", header[0], header[1], xhr), func(t *testing.T) {
				runtime := engine.NewEngine(nil)
				t.Cleanup(runtime.Close)
				value, err := runtime.RunString(fmt.Sprintf(`(async () => {
					const name = %q, value = %q;
					const url = %q + "?header=" + encodeURIComponent(name);
					if (%t) return new Promise((resolve, reject) => {
						const xhr = new XMLHttpRequest();
						xhr.open("POST", url);
						xhr.setRequestHeader(name, value);
						xhr.onload = () => resolve(xhr.responseText.trim());
						xhr.onerror = () => reject(new Error("XHR failed"));
						xhr.send("test");
					});
					const response = await fetch(url, {method: "POST", body: "test", headers: {[name]: value}});
					return (await response.text()).trim();
				})()`, header[0], header[1], server.URL, xhr))
				if err != nil {
					t.Fatal(err)
				}
				wireValue := ""
				if header[0] == "Content-Length" {
					wireValue = "4"
				}
				want, _ := json.Marshal([]string{wireValue, "test"})
				if value != string(want) {
					t.Fatalf("server received %#v, want %s", value, want)
				}
			})
		}
	}
}

func TestExtensionResponseExposesSetCookie(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Set-Cookie", "sid=one; HttpOnly")
		w.Header().Add("Set-Cookie", "token=two")
		w.Header().Set("Set-Cookie2", "legacy=three")
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)
	runtime := engine.NewEngine(nil)
	t.Cleanup(runtime.Close)
	value, err := runtime.RunString(fmt.Sprintf(`(async () => {
		const response = await fetch(%q);
		const clone = response.clone();
		await response.text(); await clone.text();
		let immutable = false;
		try { response.headers.set("Set-Cookie", "changed=1"); } catch (error) { immutable = error instanceof TypeError; }
		const synthetic = new Response(null, {headers: {"Set-Cookie": "initial=1"}});
		synthetic.headers.append("Set-Cookie", "appended=2");
		return JSON.stringify({cookies: clone.headers.getSetCookie(), legacy: response.headers.get("set-cookie2"), immutable, synthetic: synthetic.headers.getSetCookie()});
	})()`, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"cookies":["sid=one; HttpOnly","token=two"],"legacy":"legacy=three","immutable":true,"synthetic":["initial=1","appended=2"]}`
	if value != want {
		t.Fatalf("unexpected response headers: got %#v want %s", value, want)
	}
}

func TestExtensionHeadersRejectInvalidSyntax(t *testing.T) {
	runtime := engine.NewEngine(nil)
	t.Cleanup(runtime.Close)
	_, err := runtime.RunString(`(() => {
		for (const [name, value] of [["Bad Header", "ok"], ["", "ok"], ["X:Header", "ok"],
			["X-Test", "a\nb"], ["X-Test", "a\rb"], ["X-Test", "\0"], ["X-Test", "\u0100"]]) {
			for (const create of [() => new Headers([[name, value]]),
				() => new Request("http://example.com", {mode: "no-cors", headers: {[name]: value}}),
				() => { const xhr = new XMLHttpRequest(); xhr.open("GET", "http://example.com"); xhr.setRequestHeader(name, value); }]) {
				let rejected = false;
				try { create(); } catch (error) { rejected = error instanceof TypeError; }
				if (!rejected) throw new Error("Accepted invalid HTTP header syntax");
			}
		}
		return true;
	})()`)
	if err != nil {
		t.Fatal(err)
	}
}
