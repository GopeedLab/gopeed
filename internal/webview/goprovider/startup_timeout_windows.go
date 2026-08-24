//go:build cgo && windows

package goprovider

import "time"

// The first WebView2 environment initialization after a cold boot can take
// close to a minute on Windows hosted runners.
const webviewStartupTimeout = 90 * time.Second
