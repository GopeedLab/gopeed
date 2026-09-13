// Package ffmpeg runs an embedded FFmpeg with host-controlled media inputs.
package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const httpBlockSize = 1024 * 1024

// Input is a file visible to FFmpeg. Only verified HTTP Range sources seek.
type Input interface {
	io.ReadCloser
	Seek(int64, int) (int64, error)
	Size() int64
	Seekable() bool
}

var ErrSeek = errors.New("ffmpeg: input requires seeking but source is sequential")

type HTTPSource struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type httpInput struct {
	ctx                   context.Context
	client                *http.Client
	source                HTTPSource
	body                  io.ReadCloser
	size, pos, blockStart int64
	block                 []byte
	ranged                bool
	etag, modified        string
}

// OpenHTTP combines capability detection with the first media read. A 200
// response remains open and is consumed sequentially without another request.
func OpenHTTP(ctx context.Context, client *http.Client, source HTTPSource) (Input, error) {
	u, err := url.Parse(source.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("ffmpeg: expected an HTTP(S) input URL")
	}
	h := &httpInput{ctx: ctx, client: client, source: source, size: -1}
	resp, err := h.request(0, httpBlockSize-1)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusOK {
		if err = identityEncoding(resp); err != nil {
			resp.Body.Close()
			return nil, err
		}
		h.body, h.size = resp.Body, resp.ContentLength
		return h, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("ffmpeg: HTTP input returned %d", resp.StatusCode)
	}
	start, end, size, err := contentRange(resp.Header.Get("Content-Range"))
	if err != nil || start != 0 || end >= httpBlockSize {
		return nil, errors.New("ffmpeg: invalid initial Content-Range")
	}
	h.size, h.ranged = size, true
	h.etag = resp.Header.Get("ETag")
	if strings.HasPrefix(h.etag, "W/") {
		h.etag = ""
	}
	h.modified = resp.Header.Get("Last-Modified")
	h.block, err = readBlock(resp, end-start+1)
	return h, err
}

func (h *httpInput) request(start, end int64) (*http.Response, error) {
	req, err := http.NewRequestWithContext(h.ctx, http.MethodGet, h.source.URL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range h.source.Headers {
		req.Header.Set(k, v)
	}
	// Offsets must refer to the actual media bytes, not a decompressed response.
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	if h.etag != "" {
		req.Header.Set("If-Range", h.etag)
	} else if h.modified != "" {
		req.Header.Set("If-Range", h.modified)
	}
	return h.client.Do(req)
}

func identityEncoding(resp *http.Response) error {
	if enc := resp.Header.Get("Content-Encoding"); enc != "" && !strings.EqualFold(enc, "identity") {
		return errors.New("ffmpeg: encoded HTTP responses cannot represent media byte offsets")
	}
	return nil
}

func contentRange(value string) (start, end, size int64, err error) {
	err = errors.New("invalid Content-Range")
	if !strings.HasPrefix(value, "bytes ") {
		return
	}
	parts := strings.Split(strings.TrimPrefix(value, "bytes "), "/")
	if len(parts) != 2 {
		return
	}
	bounds := strings.Split(parts[0], "-")
	if len(bounds) != 2 {
		return
	}
	var e error
	if start, e = strconv.ParseInt(bounds[0], 10, 64); e != nil {
		return
	}
	if end, e = strconv.ParseInt(bounds[1], 10, 64); e != nil {
		return
	}
	if size, e = strconv.ParseInt(parts[1], 10, 64); e != nil {
		return
	}
	if start < 0 || end < start || size <= end {
		return
	}
	err = nil
	return
}

func readBlock(resp *http.Response, length int64) ([]byte, error) {
	if err := identityEncoding(resp); err != nil {
		return nil, err
	}
	if resp.ContentLength >= 0 && resp.ContentLength != length {
		return nil, errors.New("ffmpeg: Range response length mismatch")
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, length+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) != length {
		return nil, errors.New("ffmpeg: truncated or oversized Range response")
	}
	return b, nil
}

func (h *httpInput) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if err := h.ctx.Err(); err != nil {
		return 0, err
	}
	if !h.ranged {
		return h.body.Read(p)
	}
	if h.pos >= h.size {
		return 0, io.EOF
	}
	if h.pos < h.blockStart || h.pos >= h.blockStart+int64(len(h.block)) {
		end := min(h.pos+httpBlockSize-1, h.size-1)
		resp, err := h.request(h.pos, end)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusPartialContent {
			return 0, fmt.Errorf("ffmpeg: Range request returned %d; source changed or stopped supporting Range", resp.StatusCode)
		}
		start, actualEnd, size, err := contentRange(resp.Header.Get("Content-Range"))
		if err != nil || start != h.pos || actualEnd > end || size != h.size {
			return 0, errors.New("ffmpeg: inconsistent Content-Range")
		}
		if h.etag != "" && resp.Header.Get("ETag") != h.etag {
			return 0, errors.New("ffmpeg: HTTP input ETag changed")
		}
		if h.etag == "" && h.modified != "" && resp.Header.Get("Last-Modified") != h.modified {
			return 0, errors.New("ffmpeg: HTTP input Last-Modified changed")
		}
		h.block, err = readBlock(resp, actualEnd-start+1)
		if err != nil {
			return 0, err
		}
		h.blockStart = start
	}
	n := copy(p, h.block[h.pos-h.blockStart:])
	h.pos += int64(n)
	return n, nil
}

func (h *httpInput) Seek(offset int64, whence int) (int64, error) {
	if !h.ranged {
		return 0, ErrSeek
	}
	base := int64(0)
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		base = h.pos
	case io.SeekEnd:
		base = h.size
	default:
		return 0, errors.New("invalid seek origin")
	}
	next := base + offset
	if next < 0 || (offset > 0 && next < base) {
		return 0, errors.New("invalid seek offset")
	}
	h.pos = next
	return next, nil
}

func (h *httpInput) Size() int64    { return h.size }
func (h *httpInput) Seekable() bool { return h.ranged }
func (h *httpInput) Close() error {
	if h.body != nil {
		return h.body.Close()
	}
	return nil
}
