//go:build cgo && darwin

package goprovider

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework Cocoa

#include <Cocoa/Cocoa.h>

static void gopeedApplyHeadlessWindowOptions(void *windowPtr, int headless) {
  if (windowPtr == NULL) {
    return;
  }

  NSWindow *window = (__bridge NSWindow *)windowPtr;
  if (headless) {
    [window orderOut:nil];
  }
}
*/
import "C"

import (
	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
	webview "github.com/GopeedLab/webview_go"
)

func applyWindowOptions(w webview.WebView, opts enginewebview.OpenOptions) {
	C.gopeedApplyHeadlessWindowOptions(w.Window(), C.int(boolToInt(opts.Headless)))
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
