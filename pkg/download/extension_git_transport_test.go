package download

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/githubmirror"
)

func TestExtensionGitHTTPTransportAllowsSlowStreaming(t *testing.T) {
	const chunks = 24
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for range chunks {
			if _, err := io.WriteString(w, "x"); err != nil {
				return
			}
			w.(http.Flusher).Flush()
			select {
			case <-time.After(50 * time.Millisecond):
			case <-r.Context().Done():
				return
			}
		}
	}))
	defer server.Close()
	client := newExtensionGitHTTPClient(time.Second, time.Second)
	defer client.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	// A regularly streaming response lasts longer than the idle allowance.
	body, err := io.ReadAll(response.Body)
	if err != nil || len(body) != chunks {
		t.Fatalf("received %d bytes, error=%v", len(body), err)
	}
}

func TestExtensionGitHTTPTransportTimesOutStalledReads(t *testing.T) {
	for _, bodyStarted := range []bool{false, true} {
		name := "headers"
		if bodyStarted {
			name = "body"
		}
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if bodyStarted {
					_, _ = io.WriteString(w, "partial")
					w.(http.Flusher).Flush()
				}
				<-r.Context().Done()
			}))
			defer server.Close()
			client := newExtensionGitHTTPClient(time.Second, 100*time.Millisecond)
			defer client.CloseIdleConnections()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			request, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			response, err := client.Do(request)
			if err == nil {
				defer response.Body.Close()
				_, err = io.ReadAll(response.Body)
			}
			var timeout net.Error
			if !errors.As(err, &timeout) || !timeout.Timeout() || ctx.Err() != nil {
				t.Fatalf("expected idle timeout before parent deadline: error=%v context=%v", err, ctx.Err())
			}
		})
	}
}

func TestExtensionGitIdleTimeoutFallsBackThroughProxy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Hostname() == "github.com" {
			_, _ = io.WriteString(w, "partial")
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			return
		}
		_, _ = io.WriteString(w, "complete")
	}))
	defer server.Close()
	client := newExtensionGitHTTPClient(time.Second, 100*time.Millisecond)
	defer client.CloseIdleConnections()
	proxy, _ := url.Parse(server.URL)
	client.Transport.(*http.Transport).Proxy = http.ProxyURL(proxy)
	config := &githubmirror.Config{Enabled: true, Mirrors: []githubmirror.Mirror{{Type: "ghProxy", URL: server.URL}}}
	source := "http://github.com/test/repo.git"
	var tried []string
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := githubmirror.Fallback(ctx, config, source, githubmirror.Options[string]{Kind: githubmirror.Repository}, func(ctx context.Context, candidate string) (string, error) {
		tried = append(tried, candidate)
		request, _ := http.NewRequestWithContext(ctx, http.MethodGet, candidate, nil)
		response, err := client.Do(request)
		if err != nil {
			return "", err
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		return string(body), err
	})
	if err != nil || result != "complete" || !reflect.DeepEqual(tried, []string{source, server.URL + "/" + source}) {
		t.Fatalf("result=%q tried=%v error=%v", result, tried, err)
	}
}

func TestExtensionGitHTTPTransportCancelsStalledBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	client := newExtensionGitHTTPClient(time.Second, time.Minute)
	defer client.CloseIdleConnections()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	cancel()
	done := make(chan error, 1)
	go func() { _, err := io.ReadAll(response.Body); done <- err }()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("parent cancellation did not interrupt the body")
	}
}

func TestExtensionGitHTTPTransportRefreshesReusedConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			select {
			case <-time.After(500 * time.Millisecond):
			case <-r.Context().Done():
				return
			}
		}
		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()
	client := newExtensionGitHTTPClient(time.Second, time.Second)
	defer client.CloseIdleConnections()
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(700 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	reused := false
	ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{GotConn: func(info httptrace.GotConnInfo) { reused = info.Reused }})
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/slow", nil)
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || string(body) != "ok" || !reused {
		t.Fatalf("reused=%v body=%q error=%v", reused, body, err)
	}
}
