package xhr

import "github.com/imroc/req/v3"

type Fingerprint string

const (
	FingerprintMagicKey = "__gopeed_xhr_fingerprint"

	fingerprintChrome  = "chrome"
	fingerprintFirefox = "firefox"
	fingerprintSafari  = "safari"
)

func setFingerprint(client *req.Client, fingerprint string) {
	switch fingerprint {
	case fingerprintChrome:
		client.ImpersonateChrome()
	case fingerprintFirefox:
		client.ImpersonateFirefox()
	case fingerprintSafari:
		client.ImpersonateSafari()
	}
}
