//go:build cgo && webview && !windows && !darwin

package goprovider

import "testing"

func warmUpWebView(*testing.T) {}
