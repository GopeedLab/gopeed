package hls

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
)

// journalFile is the persisted completed-segment ledger. It is bound to the
// exact download plan: a journal whose fingerprint does not match the current
// plan is discarded instead of being applied to the wrong segments.
type journalFile struct {
	StagingID    string           `json:"stagingID"`
	PlanHash     string           `json:"planHash"`
	SegmentCount int              `json:"segmentCount"`
	Sizes        map[string]int64 `json:"sizes"`
}

// randomStagingID returns a random identifier used to give every task its own
// staging folder, stable across app restarts through the persisted state.
func randomStagingID() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand failing is catastrophic anyway; fall back to address
		// entropy rather than crashing the fetch.
		return fmt.Sprintf("%x", sha1.Sum([]byte(fmt.Sprintf("%p", buf))))[:12]
	}
	return hex.EncodeToString(buf)
}

// planFingerprint hashes everything that defines the download plan: the media
// URL, the ordered segments with their sequence numbers, byte ranges, keys
// and init markers. Two tasks resolve to different fingerprints even when
// they point at the same playlist URL.
func planFingerprint(state *fetcherState) string {
	h := sha1.New()
	io.WriteString(h, state.MediaURL+"\n")
	io.WriteString(h, strconv.Itoa(len(state.Segments))+"\n")
	for _, seg := range state.Segments {
		fmt.Fprintf(h, "%d|%s|", seg.Sequence, seg.URI)
		if seg.Byterange != nil {
			fmt.Fprintf(h, "%d@%d|", seg.Byterange.Offset, seg.Byterange.Length)
		}
		if seg.Key != nil {
			fmt.Fprintf(h, "key=%s|", seg.Key.URI)
		}
		if seg.IsInit {
			io.WriteString(h, "init|")
		}
		io.WriteString(h, "\n")
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// loadJournal reads the journal of a run, discarding anything that cannot be
// proven to belong to the current plan: foreign or legacy journals, corrupted
// files, and entries whose recorded size no longer matches the plan.
func loadJournal(run *hlsRun) error {
	data, err := os.ReadFile(run.journalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var file journalFile
	if err := json.Unmarshal(data, &file); err != nil {
		// A broken or legacy journal must not kill a restartable download.
		return nil
	}
	if file.StagingID != run.stagingID || file.PlanHash != run.planHash || file.SegmentCount != len(run.segs) {
		return nil
	}
	for key, size := range file.Sizes {
		idx, err := strconv.ParseInt(key, 10, 64)
		if err != nil || idx < 0 || idx >= int64(len(run.segs)) || size <= 0 {
			continue
		}
		run.journal[idx] = size
	}
	return nil
}

// saveJournal atomically rewrites the journal through a temp file + rename so
// a crash never leaves a half-written ledger behind.
func saveJournal(run *hlsRun) error {
	file := journalFile{
		StagingID:    run.stagingID,
		PlanHash:     run.planHash,
		SegmentCount: len(run.segs),
		Sizes:        make(map[string]int64, len(run.journal)),
	}
	for idx, size := range run.journal {
		file.Sizes[strconv.FormatInt(idx, 10)] = size
	}
	data, err := json.Marshal(file)
	if err != nil {
		return err
	}
	tmp := run.journalPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0666); err != nil {
		return err
	}
	return os.Rename(tmp, run.journalPath)
}
