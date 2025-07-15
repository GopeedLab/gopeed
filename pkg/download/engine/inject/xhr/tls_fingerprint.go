package xhr

import (
	"github.com/imroc/req/v3"
)

type Fingerprint string

const (
	fingerprintChrome  Fingerprint = "chrome"
	fingerprintFirefox Fingerprint = "firefox"
	fingerprintSafari  Fingerprint = "safari"
)

var currentFingerprint Fingerprint = ""

func setFingerprint(client *req.Client) {
	switch currentFingerprint {
	case fingerprintChrome:
		client.ImpersonateChrome()
	case fingerprintFirefox:
		client.ImpersonateFirefox()
	case fingerprintSafari:
		client.ImpersonateSafari()
	}
}
