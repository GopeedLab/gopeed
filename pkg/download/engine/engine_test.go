package engine

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	gojaerror "github.com/GopeedLab/gopeed/pkg/download/engine/inject/error"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/file"
	gojautil "github.com/GopeedLab/gopeed/pkg/download/engine/util"
	"github.com/dop251/goja"
)

func TestPolyfill(t *testing.T) {
	doTestPolyfill(t, "MessageError")
	doTestPolyfill(t, "XMLHttpRequest")
	doTestPolyfill(t, "Blob")
	doTestPolyfill(t, "Event")
	doTestPolyfill(t, "EventTarget")
	doTestPolyfill(t, "FormData")
	doTestPolyfill(t, "AbortController")
	doTestPolyfill(t, "AbortSignal")
	doTestPolyfill(t, "TextDecoder")
	doTestPolyfill(t, "TextEncoder")
	doTestPolyfill(t, "fetch")
	doTestPolyfill(t, "__gopeed_create_vm")
}

func TestError(t *testing.T) {
	engine := NewEngine(nil)
	_, err := engine.RunString(`
	      throw new MessageError('test');
	`)
	if me, ok := gojautil.AssertError[*gojaerror.MessageError](err); !ok {
		t.Fatalf("expect MessageError, but got %v", me)
	}
}

func TestURLCreateObjectURLRejectsUnsupportedType(t *testing.T) {
	engine := NewEngine(nil)
	defer engine.Close()

	_, err := engine.RunString(`
		URL.createObjectURL({});
	`)
	if err == nil {
		t.Fatal("expected unsupported type error")
	}
	if !strings.Contains(err.Error(), "Unsupported object type for URL.createObjectURL: Object") {
		t.Fatalf("unexpected unsupported type error: %v", err)
	}
}

func TestCallFunction_DetachedAsyncWorkDoesNotBlockReturn(t *testing.T) {
	engine := NewEngine(nil)
	defer engine.Close()
	if _, err := engine.RunString(`
		globalThis.__done = false;
		async function testDetached() {
			(async () => {
				await new Promise((resolve) => setTimeout(resolve, 500));
				globalThis.__done = true;
			})();
			return "ok";
		}
	`); err != nil {
		t.Fatal(err)
	}
	fn, ok := goja.AssertFunction(engine.Runtime.Get("testDetached"))
	if !ok {
		t.Fatal("testDetached is not callable")
	}
	startedAt := time.Now()
	value, err := engine.CallFunction(fn)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(startedAt)
	if elapsed > 200*time.Millisecond {
		t.Fatalf("detached async work blocked return for %s", elapsed)
	}
	if value != "ok" {
		t.Fatalf("unexpected return value: %#v", value)
	}
}

func TestCallFunction_AwaitedPromiseBlocksUntilSettled(t *testing.T) {
	engine := NewEngine(nil)
	defer engine.Close()
	if _, err := engine.RunString(`
		async function testAwaited() {
			await new Promise((resolve) => setTimeout(resolve, 120));
			return "ok";
		}
	`); err != nil {
		t.Fatal(err)
	}
	fn, ok := goja.AssertFunction(engine.Runtime.Get("testAwaited"))
	if !ok {
		t.Fatal("testAwaited is not callable")
	}
	startedAt := time.Now()
	value, err := engine.CallFunction(fn)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(startedAt)
	if elapsed < 100*time.Millisecond {
		t.Fatalf("awaited promise returned too early: %s", elapsed)
	}
	if value != "ok" {
		t.Fatalf("unexpected return value: %#v", value)
	}
}

func TestFetch(t *testing.T) {
	server := startServer()
	defer server.Close()
	engine := NewEngine(nil)
	if _, err := engine.RunString(fmt.Sprintf("var host = 'http://%s';", server.Addr().String())); err != nil {
		t.Fatal(err)
	}
	_, err := engine.RunString(`
async function testGet(){
	const resp = await fetch(host+'/get');
	return resp.status;
}

async function testText(){
	const resp = await fetch(host+'/text',{
		method: 'POST',
		body: 'test'
	});
	return await resp.text();
}

async function testResponseBodyGetReader() {
	const resp = await fetch(host+'/text', {
		method: 'POST',
		body: 'stream-body'
	});
	if (!resp.body || typeof resp.body.getReader !== 'function') {
		throw new Error('response.body.getReader is not available');
	}
	const reader = resp.body.getReader();
	const decoder = new TextDecoder();
	let result = '';
	while (true) {
		const { done, value } = await reader.read();
		if (done) {
			break;
		}
		result += decoder.decode(value, { stream: true });
	}
	result += decoder.decode();
	return result;
}

async function testStreamingFetchResponseBody() {
	const startedAt = Date.now();
	const resp = await fetch(host+'/stream');
	const resolvedAt = Date.now() - startedAt;
	const reader = resp.body.getReader();
	const decoder = new TextDecoder();
	const first = await reader.read();
	const firstAt = Date.now() - startedAt;
	const second = await reader.read();
	const secondAt = Date.now() - startedAt;
	let result = '';
	result += decoder.decode(first.value, { stream: true });
	result += decoder.decode(second.value, { stream: true });
	result += decoder.decode();
	return {
		resolvedAt,
		firstAt,
		secondAt,
		text: result
	};
}

async function testOctetStream(file){
	const resp = await fetch(host+'/octetStream',{
		method: 'POST',
		body: file
	});
	return await resp.text();
}

async function testRedirect() {
    const url = host + '/redirect?num=3'
    return await new Promise((resolve, reject) => {
        fetch(url, {
            method: 'HEAD',
            redirect: 'error',
        }).then(()=>reject())

        fetch(url, {
            method: 'HEAD',
            redirect: 'follow',
        }).then((res) =>res.headers.has('location') && reject()).catch(() => reject())

        fetch(url, {
            method: 'HEAD',
            redirect: 'manual',
        }).then((res) => {
			const location = res.headers.get('location');
			location ? resolve(location) : reject()
        }).catch(() => reject())
    })
}

async function testResponseUrl() {
    return new Promise((resolve, reject) => {
		const xhr = new XMLHttpRequest();
		xhr.open('GET', host+'/redirect?num=3');
		xhr.onload = function(){
            if (xhr.responseURL.includes('/redirect?num=0')){
                resolve();
            }else{
                reject();
            }
		};
		xhr.send();
	});
}

async function testFormData(file){
	const formData = new FormData();
	formData.append('name', 'test');
	formData.append('f', file);
	const resp = await fetch(host+'/formData',{
		method: 'POST',
		body: formData
	});
	return await resp.json();
}

function testHeader(){
	return new Promise((resolve, reject) => {
		const xhr = new XMLHttpRequest();
		xhr.open('GET', host+'/header');
		xhr.setRequestHeader('X-Gopeed-Test', 'test1');
		xhr.setRequestHeader('x-gopeed-test', 'test2');
		xhr.setRequestHeader('x-Gopeed-test', 'test3');
		xhr.onload = function(){
			const testHeader1 = xhr.getResponseHeader("X-Gopeed-Test");
		    const testHeader2 = xhr.getResponseHeader("x-gopeed-test");
		    const testHeader3 = xhr.getResponseHeader("x-Gopeed-test");
			const expect = 'test1, test2, test3';
			const all = xhr.getAllResponseHeaders();
			if(testHeader1 === expect && testHeader2 === expect && testHeader3 === expect 
				&& all.includes('X-Gopeed-Test: '+expect)){
				resolve();
			}else{
				reject();
			}
		};
		xhr.send();
	});
}

function testProgress(){
	return new Promise((resolve, reject) => {
		const xhr = new XMLHttpRequest();
		xhr.open('GET', host+'/get');
		const xhrUploadPromise = new Promise((resolve, reject) => {
			xhr.upload.onprogress = function(e){
				if(e.loaded === e.total){
					resolve();
				}
			}
		});
		const xhrPromise = new Promise((resolve, reject) => {
			xhr.onprogress = function(e){
				if(e.loaded === e.total){
					resolve();
				}
			}
		});
		Promise.all([xhrUploadPromise, xhrPromise]).then(() => {
			resolve();
		});
		xhr.send();
		setTimeout(() => {
			reject('timeout');
		}, 1000);
	});
}

function testAbort(){
	return new Promise((resolve, reject) => {
		const xhr = new XMLHttpRequest();
		xhr.open('GET', host+'/timeout?duration=500');
		xhr.onabort = function() {
			resolve();
		};
		xhr.send();
		setTimeout(() => {
			xhr.abort();
		}, 200);
		setTimeout(() => {
			reject('timeout');
		}, 1000);
	});
}

function testTimeout(){
	return new Promise((resolve, reject) => {
		const xhr = new XMLHttpRequest();
		const t = 500;
		xhr.open('GET', host+'/timeout?duration='+t);
		xhr.timeout = t - 200;
		xhr.onload = function() {
			resolve();
		};
		xhr.ontimeout = function() {
			reject('timeout');
		};
		xhr.send();
	});
}

async function testFingerprint(fingerprint,ua){
	__gopeed_setFingerprint(fingerprint);
	const resp = await fetch(host+'/ua');
	const data = await resp.json();
	if(!data.user_agent.includes(ua)){
		throw new Error('fingerprint test failed, user agent: ' + data.user_agent);
	}
}

async function testFingerprintDefault(){
	await testFingerprint('none', 'Go')
}

async function testFingerprintChrome(){
	await testFingerprint('chrome', 'Chrome')
}

async function testFingerprintFirefox(){
	await testFingerprint('firefox', 'Firefox')
}

async function testFingerprintSafari(){
	await testFingerprint('safari', 'Safari')
}
`)
	if err != nil {
		t.Fatal(err)
	}

	result, err := callTestFun(engine, "testGet")
	if err != nil {
		t.Fatal(err)
	}
	if result != int64(200) {
		t.Fatalf("testGet failed, want %d, got %d", 200, result)
	}

	result, err = callTestFun(engine, "testText")
	if err != nil {
		t.Fatal(err)
	}
	if result != "test" {
		t.Fatalf("testText failed, want %s, got %s", "test", result)
	}

	result, err = callTestFun(engine, "testResponseBodyGetReader")
	if err != nil {
		t.Fatal(err)
	}
	if result != "stream-body" {
		t.Fatalf("testResponseBodyGetReader failed, want %s, got %s", "stream-body", result)
	}

	result, err = callTestFun(engine, "testStreamingFetchResponseBody")
	if err != nil {
		t.Fatal(err)
	}
	streaming := result.(map[string]any)
	if streaming["text"] != "chunk-1chunk-2" {
		t.Fatalf("testStreamingFetchResponseBody failed, want %s, got %v", "chunk-1chunk-2", streaming["text"])
	}
	if int(streaming["resolvedAt"].(int64)) >= 120 {
		t.Fatalf("fetch resolved too late, got %dms", streaming["resolvedAt"].(int64))
	}
	if int(streaming["firstAt"].(int64)) >= 120 {
		t.Fatalf("first chunk arrived too late, got %dms", streaming["firstAt"].(int64))
	}
	if int(streaming["secondAt"].(int64)) < 120 {
		t.Fatalf("second chunk arrived too early, got %dms", streaming["secondAt"].(int64))
	}

	func() {
		jsFile, _, md5 := buildFile(t, engine.Runtime)
		result, err = callTestFun(engine, "testOctetStream", jsFile)
		if err != nil {
			t.Fatal(err)
		}
		if result != md5 {
			t.Fatalf("testOctetStream failed, want %s, got %s", md5, result)
		}
	}()

	t.Run("testRedirect", func(t *testing.T) {
		_, err := callTestFun(engine, "testRedirect")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("testResponseUrl", func(t *testing.T) {
		_, err = callTestFun(engine, "testResponseUrl")
		if err != nil {
			t.Fatal(err)
		}
	})

	func() {
		jsFile, goFile, md5 := buildFile(t, engine.Runtime)
		result, err = callTestFun(engine, "testFormData", jsFile)
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]any{
			"name": "test",
			"f": map[string]string{
				"filename": goFile.Name,
				"md5":      md5,
			},
		}
		if !test.JsonEqual(result, want) {
			t.Fatalf("testFormData failed, want %v, got %v", want, result)
		}
	}()

	_, err = callTestFun(engine, "testHeader")
	if err != nil {
		t.Fatal("header test failed", err)
	}

	_, err = callTestFun(engine, "testProgress")
	if err != nil {
		t.Fatal("progress test failed", err)
	}

	_, err = callTestFun(engine, "testAbort")
	if err != nil {
		t.Fatal("abort test failed", err)
	}

	_, err = callTestFun(engine, "testTimeout")
	if err == nil || err.Error() != "timeout" {
		t.Fatalf("timeout test failed, want %s, got %s", "timeout", err)
	}

	_, err = callTestFun(engine, "testFingerprintChrome")
	if err != nil {
		t.Fatal("testFingerprintChrome test failed", err)
	}
	_, err = callTestFun(engine, "testFingerprintFirefox")
	if err != nil {
		t.Fatal("testFingerprintFirefox test failed", err)
	}
	_, err = callTestFun(engine, "testFingerprintSafari")
	if err != nil {
		t.Fatal("testFingerprintSafari test failed", err)
	}
}

func TestFetchWithProxy(t *testing.T) {
	doTestFetchWithProxy(t, "", "")
	doTestFetchWithProxy(t, "admin", "123")
}

func doTestFetchWithProxy(t *testing.T, usr, pwd string) {
	httpListener := startServer()
	defer httpListener.Close()

	proxyListener := test.StartSocks5Server(usr, pwd)
	defer proxyListener.Close()
	engine := NewEngine(&Config{
		ProxyConfig: &base.DownloaderProxyConfig{
			Enable: true,
			System: false,
			Scheme: "socks5",
			Host:   proxyListener.Addr().String(),
			Usr:    usr,
			Pwd:    pwd,
		},
	})

	if _, err := engine.RunString(fmt.Sprintf("var host = 'http://%s';", httpListener.Addr().String())); err != nil {
		t.Fatal(err)
	}

	respCode, err := engine.RunString(`
(async function(){
	const resp = await fetch(host+'/get');
	return resp.status;
})()
`)
	if err != nil {
		t.Fatal(err)
	}
	if respCode != int64(200) {
		t.Fatalf("fetch with proxy failed, want %d, got %d", 200, respCode)
	}
}

func TestVm(t *testing.T) {
	engine := NewEngine(nil)

	value, err := engine.RunString(`
const vm = __gopeed_create_vm()
vm.set('a', 1)
vm.set('b', 2)
const result = vm.runString('a=a+1;b=b+1;a+b;')
const out = {
	"a": vm.get('a'),
	"b": vm.get('b'),
	"result": result
}
out
`)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"a":      2,
		"b":      3,
		"result": 5,
	}
	if !test.JsonEqual(value, want) {
		t.Fatalf("vm test failed, want %v, got %v", want, value)
	}
}

func TestNonStopLoop(t *testing.T) {
	engine := NewEngine(nil)

	_, err := engine.RunString(`
function leak(){
	setInterval(() => {
	},500)
}

function test(){
	leak()
	return new Promise((resolve, reject) => {
		setTimeout(() => {
			resolve('done')
		}, 1000)	
	})
}
`)
	if err != nil {
		t.Fatal(err)
	}

	val, err := callTestFun(engine, "test")
	if err != nil {
		panic(err)
	}
	if val != "done" {
		t.Fatalf("infinite loop test failed, want %s, got %s", "done", val)
	}
}

func doTestPolyfill(t *testing.T, module string) {
	value, err := Run(fmt.Sprintf(`
!!globalThis['%s']
`, module))
	if err != nil {
		t.Fatal(err)
	}
	if !value.(bool) {
		t.Fatalf("module %s not polyfilled", module)
	}
}

func startServer() net.Listener {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	server := &http.Server{}
	mux := http.NewServeMux()
	mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/header", func(w http.ResponseWriter, r *http.Request) {
		for k, v := range r.Header {
			if strings.HasPrefix(k, "X-Gopeed") {
				w.Header().Set(k, strings.Join(v, ", "))
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/text", func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write(buf)
	})
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			_, _ = w.Write([]byte("chunk-1"))
			flusher.Flush()
			time.Sleep(150 * time.Millisecond)
			_, _ = w.Write([]byte("chunk-2"))
			flusher.Flush()
			return
		}
		_, _ = w.Write([]byte("chunk-1chunk-2"))
	})
	mux.HandleFunc("/octetStream", func(w http.ResponseWriter, r *http.Request) {
		md5 := calcMd5(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(md5))
	})
	mux.HandleFunc("/formData", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(1024 * 1024 * 30)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		result := make(map[string]any)
		for k, v := range r.MultipartForm.Value {
			result[k] = v[0]
		}
		for k, v := range r.MultipartForm.File {
			f, _ := v[0].Open()
			result[k] = map[string]string{
				"filename": v[0].Filename,
				"md5":      calcMd5(f),
			}
		}
		w.WriteHeader(http.StatusOK)
		buf, _ := json.Marshal(result)
		w.Write(buf)
	})
	mux.HandleFunc("/timeout", func(w http.ResponseWriter, r *http.Request) {
		duration := r.URL.Query().Get("duration")
		t, _ := strconv.Atoi(duration)
		time.Sleep(time.Duration(t) * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		num := r.URL.Query().Get("num")
		n, _ := strconv.Atoi(num)
		if n > 0 {
			http.Redirect(w, r, fmt.Sprintf("/redirect?num=%d", n-1), http.StatusFound)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}
	})
	mux.HandleFunc("/ua", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data := map[string]any{
			"user_agent": r.UserAgent(),
		}
		buf, _ := json.Marshal(data)
		w.WriteHeader(http.StatusOK)
		w.Write(buf)
	})
	server.Handler = mux
	go server.Serve(listener)
	return listener
}

func buildFile(t *testing.T, runtime *goja.Runtime) (goja.Value, *file.File, string) {
	jsFile, err := file.NewJsFile(runtime)
	if err != nil {
		t.Fatal(err)
	}
	f := jsFile.Export().(*file.File)
	data := "test"
	f.Reader = strings.NewReader(data)
	f.Name = "test.txt"
	f.Size = int64(len(data))
	return jsFile, f, calcMd5(strings.NewReader(data))
}

func callTestFun(engine *Engine, fun string, args ...any) (any, error) {
	test, ok := goja.AssertFunction(engine.Runtime.Get(fun))
	if !ok {
		return nil, errors.New("function not found:" + fun)
	}
	return engine.CallFunction(test, args...)
}

func calcMd5(reader io.Reader) string {
	// Open a new hash interface to write to
	hash := md5.New()

	// Copy the file in the hash interface and check for any error
	if _, err := io.Copy(hash, reader); err != nil {
		return ""
	}
	return hex.EncodeToString(hash.Sum(nil))
}
