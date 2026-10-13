package engine_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/download/engine"
)

func TestFetchDoesNotDependOnXMLHttpRequest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/target", func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("X-Method", request.Method)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, request *http.Request) {
		http.Redirect(w, request, "/target", http.StatusFound)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	runtime := engine.NewEngine(nil)
	t.Cleanup(runtime.Close)

	value, err := runtime.RunString(fmt.Sprintf(`
		(async () => {
			globalThis.XMLHttpRequest = function () {
				throw new Error("fetch used XMLHttpRequest");
			};

			const get = await fetch(%q);
			const head = await fetch(%q, { method: "HEAD" });
			const manual = await fetch(%q, { redirect: "manual" });
			let redirectError = false;
			try {
				await fetch(%q, { redirect: "error" });
			} catch (_) {
				redirectError = true;
			}

			return JSON.stringify({
				get: await get.text(),
				headStatus: head.status,
				headMethod: head.headers.get("x-method"),
				manualStatus: manual.status,
				manualLocation: manual.headers.get("location"),
				redirectError,
			});
		})()
	`, server.URL+"/target", server.URL+"/target", server.URL+"/redirect", server.URL+"/redirect"))
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"get":"ok","headStatus":200,"headMethod":"HEAD","manualStatus":302,"manualLocation":"/target","redirectError":true}`
	if value != want {
		t.Fatalf("unexpected fetch result: got %#v want %s", value, want)
	}
}

func TestXMLHttpRequestUsesFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Test", "value")
		_, _ = w.Write([]byte("payload"))
	}))
	t.Cleanup(server.Close)

	runtime := engine.NewEngine(nil)
	t.Cleanup(runtime.Close)

	value, err := runtime.RunString(fmt.Sprintf(`
		(async () => {
			const originalFetch = globalThis.fetch;
			let fetchCalls = 0;
			globalThis.fetch = function (...args) {
				fetchCalls++;
				return originalFetch(...args);
			};

			const result = await new Promise((resolve, reject) => {
				const xhr = new XMLHttpRequest();
				const states = [];
				xhr.onreadystatechange = () => states.push(xhr.readyState);
				xhr.onerror = () => reject(new Error("XHR failed"));
				xhr.onload = () => resolve({
					status: xhr.status,
					text: xhr.responseText,
					header: xhr.getResponseHeader("x-test"),
					states,
				});
				xhr.open("GET", %q);
				xhr.send();
			});
			result.fetchCalls = fetchCalls;
			return JSON.stringify(result);
		})()
	`, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"status":200,"text":"payload","header":"value","states":[1,2,3,4],"fetchCalls":1}`
	if value != want {
		t.Fatalf("unexpected XMLHttpRequest result: got %#v want %s", value, want)
	}
}

func TestExtensionRequestsSendExplicitCookie(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		_, _ = w.Write([]byte(request.Header.Get("Cookie")))
	}))
	t.Cleanup(server.Close)

	type requestTest struct {
		name   string
		script string
	}
	tests := []requestTest{
		{"fetch record", `const response = await fetch(url, {headers: {Cookie: cookie}}); return response.text();`},
		{"fetch Headers", `const response = await fetch(url, {headers: new Headers({cookie})}); return response.text();`},
		{"fetch pairs", `const response = await fetch(url, {headers: [["cOoKiE", cookie]]}); return response.text();`},
		{"fetch repeated Cookie", `const headers = new Headers([["Cookie", "sid=manual"], ["cookie", "token=provided"]]); if (headers.get("cookie") !== cookie) throw new Error("Incorrect Cookie separator"); const response = await fetch(url, {headers}); return response.text();`},
		{"fetch Request clone", `const request = new Request(url, {headers: {Cookie: cookie}}); const response = await fetch(request.clone()); return response.text();`},
		{"fetch Request set", `const request = new Request(url); request.headers.set("Cookie", cookie); const response = await fetch(request); return response.text();`},
		{"fetch Request append", `const request = new Request(url); request.headers.append("Cookie", cookie); const response = await fetch(request); return response.text();`},
		{"fetch no-cors mutations", `const request = new Request(url, {mode: "no-cors"}); request.headers.set("Cookie", "removed=1"); request.headers.delete("COOKIE"); if (request.headers.has("cookie")) throw new Error("Cookie was not deleted"); request.headers.append("Cookie", cookie); const response = await fetch(request.clone()); return response.text();`},
	}
	for _, mode := range []string{"cors", "same-origin", "no-cors"} {
		for _, credentials := range []string{"omit", "same-origin", "include"} {
			tests = append(tests, requestTest{
				name:   "fetch " + mode + " " + credentials,
				script: fmt.Sprintf(`const response = await fetch(url, {headers: {Cookie: cookie}, mode: %q, credentials: %q}); return response.text();`, mode, credentials),
			})
		}
	}
	for _, withCredentials := range []bool{false, true} {
		tests = append(tests, requestTest{
			name: fmt.Sprintf("XHR withCredentials=%t", withCredentials),
			script: fmt.Sprintf(`return new Promise((resolve, reject) => {
				const xhr = new XMLHttpRequest();
				xhr.open("GET", url);
				xhr.withCredentials = %t;
				xhr.setRequestHeader("cOoKiE", cookie);
				xhr.onload = () => resolve(xhr.responseText);
				xhr.onerror = () => reject(new Error("XHR failed"));
				xhr.send();
			});`, withCredentials),
		})
	}
	tests = append(tests, requestTest{"XHR repeated Cookie", `return new Promise((resolve, reject) => {
		const xhr = new XMLHttpRequest();
		xhr.open("GET", url);
		xhr.setRequestHeader("Cookie", "sid=manual");
		xhr.setRequestHeader("cookie", "token=provided");
		xhr.onload = () => resolve(xhr.responseText);
		xhr.onerror = () => reject(new Error("XHR failed"));
		xhr.send();
	});`})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runtime := engine.NewEngine(nil)
			t.Cleanup(runtime.Close)
			const cookie = "sid=manual; token=provided"
			value, err := runtime.RunString(fmt.Sprintf(`(async () => {
				const url = %q;
				const cookie = %q;
				%s
			})()`, server.URL, cookie, test.script))
			if err != nil {
				t.Fatal(err)
			}
			if value != cookie {
				t.Fatalf("server received Cookie %q, want %q", value, cookie)
			}
		})
	}
}

func TestFetchExplicitCookieWithCookieJar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/seed" {
			http.SetCookie(w, &http.Cookie{Name: "stored", Value: "jar", Path: "/"})
		}
		_, _ = w.Write([]byte(request.Header.Get("Cookie")))
	}))
	t.Cleanup(server.Close)
	runtime := engine.NewEngine(nil)
	t.Cleanup(runtime.Close)
	value, err := runtime.RunString(fmt.Sprintf(`(async () => {
		const url = %q;
		await (await fetch(url + "/seed")).text();
		const automatic = await (await fetch(url)).text();
		const combined = await (await fetch(url, {headers: {Cookie: "manual=provided"}})).text();
		const omitted = await (await fetch(url, {credentials: "omit", headers: {Cookie: "manual=provided"}})).text();
		const empty = await (await fetch(url, {credentials: "omit"})).text();
		return JSON.stringify({automatic, combined, omitted, empty});
	})()`, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"automatic":"stored=jar","combined":"manual=provided; stored=jar","omitted":"manual=provided","empty":""}`
	if value != want {
		t.Fatalf("unexpected CookieJar result: got %#v want %s", value, want)
	}
}

func TestFetchResponseAndRequestCloneHaveIndependentBodies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPost {
			_, _ = response.Write([]byte("posted"))
			return
		}
		_, _ = response.Write([]byte("payload"))
	}))
	t.Cleanup(server.Close)

	runtime := engine.NewEngine(nil)
	t.Cleanup(runtime.Close)

	value, err := runtime.RunString(fmt.Sprintf(`
		(async () => {
			const response = await fetch(%q);
			const responseClone = response.clone();
			const responseBodies = await Promise.all([response.text(), responseClone.text()]);

			const request = new Request(%q, { method: "POST", body: "request-body" });
			const requestClone = request.clone();
			const requestBodies = await Promise.all([request.text(), requestClone.text()]);

			return JSON.stringify({ responseBodies, requestBodies });
		})()
	`, server.URL, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"responseBodies":["payload","payload"],"requestBodies":["request-body","request-body"]}`
	if value != want {
		t.Fatalf("unexpected clone result: got %#v want %s", value, want)
	}
}

func TestFetchFormDataUsesWebEntryListSemantics(t *testing.T) {
	runtime := engine.NewEngine(nil)
	t.Cleanup(runtime.Close)

	value, err := runtime.RunString(`
		(() => {
			const data = new FormData();
			data.append("a", "one");
			data.append("a", 2);
			data.append("b", "three");
			const beforeSet = Array.from(data.entries());
			data.set("a", "four");
			const afterSet = Array.from(data);
			const all = data.getAll("a");
			data.delete("b");
			return JSON.stringify({ beforeSet, afterSet, all, hasB: data.has("b") });
		})()
	`)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"beforeSet":[["a","one"],["a","2"],["b","three"]],"afterSet":[["a","four"],["b","three"]],"all":["four"],"hasB":false}`
	if value != want {
		t.Fatalf("unexpected FormData result: got %#v want %s", value, want)
	}
}
