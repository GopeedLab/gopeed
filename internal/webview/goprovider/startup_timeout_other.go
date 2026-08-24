//go:build cgo && !windows

package goprovider

import "time"

const webviewStartupTimeout = 10 * time.Second
