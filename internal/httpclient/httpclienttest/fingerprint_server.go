package httpclienttest

import (
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type Profile string

const (
	ProfileNative     Profile = "native"
	ProfileChrome     Profile = "chrome"
	ProfileFirefox    Profile = "firefox"
	ProfileSafari     Profile = "safari"
	ProfileAnyBrowser Profile = "browser"
)

type FingerprintServerOptions struct {
	RequiredProfile Profile
	Handler         http.Handler
}

type FingerprintServer struct {
	*httptest.Server

	mu       sync.Mutex
	profiles []Profile
}

func NewFingerprintServer(t testing.TB, options FingerprintServerOptions) *FingerprintServer {
	t.Helper()

	handler := options.Handler
	if handler == nil {
		handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}
	server := &FingerprintServer{
		Server: httptest.NewUnstartedServer(handler),
	}
	server.EnableHTTP2 = true
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.TLS = &tls.Config{
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			profile := classifyProfile(hello.CipherSuites)
			server.mu.Lock()
			server.profiles = append(server.profiles, profile)
			server.mu.Unlock()
			if profileMatches(profile, options.RequiredProfile) {
				return nil, nil
			}
			return nil, errors.New("required TLS fingerprint was not presented")
		},
	}
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}

func (s *FingerprintServer) Profiles() []Profile {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Profile(nil), s.profiles...)
}

func profileMatches(actual, required Profile) bool {
	if required == ProfileAnyBrowser {
		return actual != ProfileNative
	}
	return actual == required
}

func classifyProfile(cipherSuites []uint16) Profile {
	if len(cipherSuites) >= 5 && isGREASE(cipherSuites[0]) {
		switch cipherSuites[4] {
		case tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256:
			return ProfileChrome
		case tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384:
			return ProfileSafari
		}
	}
	if len(cipherSuites) >= 3 &&
		cipherSuites[0] == tls.TLS_AES_128_GCM_SHA256 &&
		cipherSuites[1] == tls.TLS_CHACHA20_POLY1305_SHA256 &&
		cipherSuites[2] == tls.TLS_AES_256_GCM_SHA384 {
		return ProfileFirefox
	}
	return ProfileNative
}

func isGREASE(value uint16) bool {
	return value&0x0f0f == 0x0a0a && byte(value) == byte(value>>8)
}
