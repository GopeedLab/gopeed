package http

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type ReqExtra struct {
	Method string            `json:"method"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
}

// ChecksumOption specifies an expected checksum to verify against
// the completed download. Only supported for HTTP downloads.
type ChecksumOption struct {
	Algorithm string `json:"algorithm"` // "md5" | "sha1" | "sha256"
	Expected  string `json:"expected"`  // hex-encoded expected hash
}

// IsEmpty returns true when chk is nil, or when both Algorithm and Expected
// are blank after trimming whitespace.
func (chk *ChecksumOption) IsEmpty() bool {
	if chk == nil {
		return true
	}
	return strings.TrimSpace(chk.Algorithm) == "" && strings.TrimSpace(chk.Expected) == ""
}

// NormalizedAlgorithm returns the canonical lower-cased algorithm ("md5", "sha1", "sha256")
// and true if recognized, or ("", false) if unsupported.
func (chk *ChecksumOption) NormalizedAlgorithm() (string, bool) {
	if chk == nil {
		return "", false
	}
	switch strings.ToLower(strings.TrimSpace(chk.Algorithm)) {
	case "md5":
		return "md5", true
	case "sha1", "sha-1":
		return "sha1", true
	case "sha256", "sha-256":
		return "sha256", true
	default:
		return "", false
	}
}

// Validate checks whether the checksum option is valid.
// Empty options return nil (no verification).
func (chk *ChecksumOption) Validate() error {
	if chk.IsEmpty() {
		return nil
	}
	algo, ok := chk.NormalizedAlgorithm()
	if !ok {
		return fmt.Errorf("unsupported checksum algorithm: %s", chk.Algorithm)
	}
	expected := strings.TrimSpace(chk.Expected)
	if expected == "" {
		return fmt.Errorf("checksum expected hash is required when algorithm is set")
	}
	var expectedLen int
	switch algo {
	case "md5":
		expectedLen = 32
	case "sha1":
		expectedLen = 40
	case "sha256":
		expectedLen = 64
	}
	if len(expected) != expectedLen {
		return fmt.Errorf("invalid checksum hash length for %s: expected %d characters, got %d", algo, expectedLen, len(expected))
	}
	if _, err := hex.DecodeString(expected); err != nil {
		return fmt.Errorf("invalid checksum hash for %s: expected valid hexadecimal string", algo)
	}
	return nil
}

type OptsExtra struct {
	Connections int `json:"connections"`
	// AutoTorrent when task download complete, and it is a .torrent file, it will be auto create a new task for the torrent file
	// nil means use global config, true/false means explicit setting
	AutoTorrent *bool `json:"autoTorrent"`
	// DeleteTorrentAfterDownload when true, deletes the .torrent file after creating BT task
	// nil means use global config, true/false means explicit setting
	DeleteTorrentAfterDownload *bool `json:"deleteTorrentAfterDownload"`
	// AutoExtract when task download complete, and it is an archive file, it will be auto extracted
	// nil means use global config, true/false means explicit setting
	AutoExtract *bool `json:"autoExtract"`
	// ArchivePassword is the password for extracting password-protected archives
	ArchivePassword string `json:"archivePassword"`
	// DeleteAfterExtract when true, deletes the archive file after successful extraction
	DeleteAfterExtract bool `json:"deleteAfterExtract"`
	// Checksum verifies the completed file against an expected hash after download
	Checksum *ChecksumOption `json:"checksum,omitempty"`
}

// Stats for download
type Stats struct {
	Connections []*StatsConnection `json:"connections"`
}

type StatsConnection struct {
	Downloaded int64 `json:"downloaded"`
	// Total is the resource size divided by the current number of connections,
	// rounded up to a whole byte. Every connection in one snapshot uses the same
	// denominator so clients can compare their downloaded contributions. It is
	// zero when the resource size is unknown.
	Total      int64 `json:"total"`
	Completed  bool  `json:"completed"`
	Failed     bool  `json:"failed"`
	RetryTimes int   `json:"retryTimes"`
}
