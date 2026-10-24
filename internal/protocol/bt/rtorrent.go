package bt

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/GopeedLab/gopeed/internal/xmlrpc"
)

// rtorrentClient is the part of rtorrent's XML-RPC API that the hand-off
// uses. Info hashes are 40 uppercase hex digits, the way rtorrent names its
// downloads. Every value that comes from a user, such as a folder, goes in as
// a string parameter of its own and never into rtorrent's command text.
type rtorrentClient struct {
	rpc *xmlrpc.Client
}

func newRtorrentClient(cfg *externalConfig) (*rtorrentClient, error) {
	if cfg == nil || cfg.URL == "" {
		return nil, errors.New("the rtorrent address is not set")
	}
	rpc, err := xmlrpc.NewClient(cfg.URL, cfg.Username, cfg.Password)
	if err != nil {
		return nil, fmt.Errorf("rtorrent %w", err)
	}
	return &rtorrentClient{rpc: rpc}, nil
}

func (c *rtorrentClient) call(ctx context.Context, method string, params ...any) (any, error) {
	v, err := c.rpc.Call(ctx, method, params...)
	if err != nil {
		return nil, fmt.Errorf("rtorrent %w", err)
	}
	return v, nil
}

// rtorrentStatus is what one poll reads about a download.
type rtorrentStatus struct {
	Name           string
	Meta           bool // a magnet that still waits for its metadata
	Started        bool // d.state: started by the user, not stopped
	Active         bool
	Hashing        bool
	Complete       bool
	CompletedBytes int64
	SizeBytes      int64
	WantedChunks   int64
	SizeChunks     int64
	DownRate       int64
	UpRate         int64
	UpTotal        int64
	Ratio          int64 // per mille
	Peers          int64
	Seeders        int64
	Message        string
	Directory      string
	MultiFile      bool
}

// done reports whether every selected file is complete. With some files
// turned off rtorrent never reports the whole download complete; it reports
// no wanted chunks left instead.
func (s *rtorrentStatus) done() bool {
	if s.Meta || s.Hashing || s.SizeChunks == 0 {
		return false
	}
	return s.Complete || (s.Started && s.WantedChunks == 0)
}

// rtorrentFile is one file of a download.
type rtorrentFile struct {
	Path            string
	SizeBytes       int64
	CompletedChunks int64
	SizeChunks      int64
}

// completedBytes estimates the bytes of the file on disk from its chunks.
// A chunk at a file boundary is shared with the neighbour, so the estimate
// can be a little off until the file is complete.
func (f *rtorrentFile) completedBytes() int64 {
	if f.SizeChunks <= 0 || f.CompletedChunks <= 0 {
		return 0
	}
	if f.CompletedChunks >= f.SizeChunks {
		return f.SizeBytes
	}
	return f.SizeBytes * f.CompletedChunks / f.SizeChunks
}

// has reports whether rtorrent holds the download. rtorrent answers an
// unknown hash with a fault.
func (c *rtorrentClient) has(ctx context.Context, hash string) (bool, error) {
	_, err := c.call(ctx, "d.hash", hash)
	if err == nil {
		return true, nil
	}
	var fault *xmlrpc.Fault
	if errors.As(err, &fault) {
		return false, nil
	}
	return false, err
}

// loadRaw adds a torrent from its bytes, stopped.
func (c *rtorrentClient) loadRaw(ctx context.Context, data []byte) error {
	_, err := c.call(ctx, "load.raw", "", data)
	return err
}

// loadMagnet adds a magnet link, stopped. rtorrent then holds a small
// metadata download under the same hash until the swarm sends the metadata.
func (c *rtorrentClient) loadMagnet(ctx context.Context, uri string) error {
	_, err := c.call(ctx, "load.normal", "", uri)
	return err
}

// waitLoaded waits for a download that was just loaded to appear.
func (c *rtorrentClient) waitLoaded(ctx context.Context, hash string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		ok, err := c.has(ctx, hash)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("rtorrent did not add the torrent; check its log, and network.xmlrpc.size_limit for a large torrent")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// setDirectory sets where the download is saved. For a torrent with a root
// folder rtorrent saves into <dir>/<name>, the same layout as the built-in
// engine. It only works while the download is closed.
func (c *rtorrentClient) setDirectory(ctx context.Context, hash, dir string) error {
	_, err := c.call(ctx, "d.directory.set", hash, dir)
	return err
}

// setSelection turns the files that are not in selected off and the others
// on. A nil selection turns every file on.
func (c *rtorrentClient) setSelection(ctx context.Context, hash string, fileCount int, selected []int) error {
	want := make(map[int]bool, len(selected))
	for _, i := range selected {
		want[i] = true
	}
	calls := make([]any, 0, fileCount+1)
	for i := 0; i < fileCount; i++ {
		priority := 1
		if selected != nil && !want[i] {
			priority = 0
		}
		calls = append(calls, map[string]any{
			"methodName": "f.priority.set",
			"params":     []any{hash + ":f" + strconv.Itoa(i), priority},
		})
	}
	calls = append(calls, map[string]any{"methodName": "d.update_priorities", "params": []any{hash}})
	results, err := c.multicall(ctx, calls)
	if err != nil {
		return err
	}
	for _, r := range results {
		if r.err != nil {
			return r.err
		}
	}
	return nil
}

func (c *rtorrentClient) start(ctx context.Context, hash string) error {
	_, err := c.call(ctx, "d.start", hash)
	return err
}

func (c *rtorrentClient) stop(ctx context.Context, hash string) error {
	_, err := c.call(ctx, "d.stop", hash)
	return err
}

// erase removes the download from rtorrent. rtorrent never deletes the data.
func (c *rtorrentClient) erase(ctx context.Context, hash string) error {
	_, err := c.call(ctx, "d.erase", hash)
	return err
}

// rtorrentStatusFields is the order of the calls status makes.
var rtorrentStatusFields = []string{
	"d.name", "d.is_meta", "d.state", "d.is_active", "d.hashing", "d.complete",
	"d.completed_bytes", "d.size_bytes", "d.wanted_chunks", "d.size_chunks",
	"d.down.rate", "d.up.rate", "d.up.total", "d.ratio", "d.peers_connected",
	"d.peers_complete", "d.message", "d.directory", "d.is_multi_file",
}

// status reads the download in one round trip.
func (c *rtorrentClient) status(ctx context.Context, hash string) (*rtorrentStatus, error) {
	calls := make([]any, len(rtorrentStatusFields))
	for i, m := range rtorrentStatusFields {
		calls[i] = map[string]any{"methodName": m, "params": []any{hash}}
	}
	results, err := c.multicall(ctx, calls)
	if err != nil {
		return nil, err
	}
	if len(results) != len(rtorrentStatusFields) {
		return nil, fmt.Errorf("rtorrent answered %d of %d status fields", len(results), len(rtorrentStatusFields))
	}
	// The first call names the download; a fault there means rtorrent no
	// longer has it.
	if results[0].err != nil {
		return nil, errRtorrentGone
	}
	v := func(i int) any { return results[i].value }
	s := &rtorrentStatus{
		Name:           asString(v(0)),
		Meta:           asInt(v(1)) == 1,
		Started:        asInt(v(2)) == 1,
		Active:         asInt(v(3)) == 1,
		Hashing:        asInt(v(4)) != 0,
		Complete:       asInt(v(5)) == 1,
		CompletedBytes: asInt(v(6)),
		SizeBytes:      asInt(v(7)),
		WantedChunks:   asInt(v(8)),
		SizeChunks:     asInt(v(9)),
		DownRate:       asInt(v(10)),
		UpRate:         asInt(v(11)),
		UpTotal:        asInt(v(12)),
		Ratio:          asInt(v(13)),
		Peers:          asInt(v(14)),
		Seeders:        asInt(v(15)),
		Message:        asString(v(16)),
		Directory:      asString(v(17)),
		MultiFile:      asInt(v(18)) == 1,
	}
	if s.Meta {
		// A metadata download reports a size of one byte.
		s.SizeBytes, s.CompletedBytes = 0, 0
	}
	return s, nil
}

// files lists the files of a download, in torrent order.
func (c *rtorrentClient) files(ctx context.Context, hash string) ([]rtorrentFile, error) {
	v, err := c.call(ctx, "f.multicall", hash, "", "f.path=", "f.size_bytes=", "f.completed_chunks=", "f.size_chunks=")
	if err != nil {
		return nil, err
	}
	rows, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("rtorrent f.multicall: unexpected answer %T", v)
	}
	files := make([]rtorrentFile, 0, len(rows))
	for _, row := range rows {
		cols, ok := row.([]any)
		if !ok || len(cols) < 4 {
			return nil, errors.New("rtorrent f.multicall: unexpected row")
		}
		files = append(files, rtorrentFile{
			Path:            asString(cols[0]),
			SizeBytes:       asInt(cols[1]),
			CompletedChunks: asInt(cols[2]),
			SizeChunks:      asInt(cols[3]),
		})
	}
	return files, nil
}

type multicallResult struct {
	value any
	err   error
}

// multicall runs several calls in one request with system.multicall. Each
// result is the call's value or its own fault.
func (c *rtorrentClient) multicall(ctx context.Context, calls []any) ([]multicallResult, error) {
	v, err := c.call(ctx, "system.multicall", calls)
	if err != nil {
		return nil, err
	}
	list, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("rtorrent system.multicall: unexpected answer %T", v)
	}
	out := make([]multicallResult, len(list))
	for i, item := range list {
		switch t := item.(type) {
		case []any:
			if len(t) > 0 {
				out[i].value = t[0]
			}
		case map[string]any:
			f := &xmlrpc.Fault{}
			if code, ok := t["faultCode"].(int64); ok {
				f.Code = int(code)
			}
			f.Message, _ = t["faultString"].(string)
			out[i].err = f
		default:
			out[i].err = fmt.Errorf("rtorrent system.multicall: unexpected result %T", item)
		}
	}
	return out, nil
}

var errRtorrentGone = errors.New("rtorrent no longer has this torrent")

func asInt(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case bool:
		if t {
			return 1
		}
	case string:
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	}
	return 0
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
