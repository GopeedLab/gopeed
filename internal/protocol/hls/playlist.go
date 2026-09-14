package hls

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
)

// Variant is one rendition entry of a master playlist.
type Variant struct {
	URI          string
	Bandwidth    int64
	AvgBandwidth int64
	Resolution   string
	Codecs       string
	Name         string
}

// Key is the decryption key applied to the segments that follow its tag, until
// the next EXT-X-KEY tag or the end of the playlist.
type Key struct {
	Method string
	URI    string // absolute URI
	IV     []byte // explicit IV; nil means derive from the media sequence number
}

// ByteRange is an EXT-X-BYTERANGE sub range of the segment resource.
type ByteRange struct {
	Length int64
	Offset int64
}

// Segment is one downloadable piece of a media playlist. The init section of
// an fMP4 stream is represented as a segment with IsInit set.
type Segment struct {
	URI       string
	Sequence  int64 // media sequence number, used to derive the default AES IV
	Byterange *ByteRange
	Key       *Key // nil means unencrypted
	IsInit    bool
	Gap       bool
	Duration  float64
	Size      int64 // exact size in bytes when known (byterange / HEAD probe), 0 when unknown
}

// Media is a parsed media playlist.
type Media struct {
	Segments []*Segment
	Version  int
	Live     bool // no EXT-X-ENDLIST tag
	IsFMP4   bool // playlist declares an init section via EXT-X-MAP
}

const (
	keyMethodNone      = "NONE"
	keyMethodAES128    = "AES-128"
	keyMethodSampleAES = "SAMPLE-AES"
)

// IsMasterPlaylist reports whether the content looks like a master playlist.
func IsMasterPlaylist(content string) bool {
	return strings.Contains(content, "#EXT-X-STREAM-INF")
}

func validateM3U8Header(content string) error {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line != "#EXTM3U" {
			return errors.New("content is not a valid m3u8 playlist")
		}
		return nil
	}
	return errors.New("empty m3u8 playlist")
}

// ParseMaster parses a master playlist and returns all variants.
func ParseMaster(content string, baseURL *url.URL) ([]*Variant, error) {
	if err := validateM3U8Header(content); err != nil {
		return nil, err
	}
	var variants []*Variant
	var pending *Variant
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {
			attrs := parseAttributes(line[len("#EXT-X-STREAM-INF:"):])
			pending = &Variant{
				Bandwidth:    parseInt64(attrs["BANDWIDTH"]),
				AvgBandwidth: parseInt64(attrs["AVERAGE-BANDWIDTH"]),
				Resolution:   attrs["RESOLUTION"],
				Codecs:       unquote(attrs["CODECS"]),
				Name:         unquote(attrs["NAME"]),
			}
			continue
		}
		if pending != nil {
			if line[0] != '#' {
				pending.URI = resolveURL(baseURL, line)
				variants = append(variants, pending)
			}
			pending = nil
		}
	}
	if len(variants) == 0 {
		return nil, errors.New("master playlist has no variants")
	}
	return variants, nil
}

// PickBestVariant selects the highest-bandwidth variant.
func PickBestVariant(variants []*Variant) *Variant {
	var best *Variant
	for _, v := range variants {
		if best == nil || v.Bandwidth > best.Bandwidth {
			best = v
		}
	}
	return best
}

// ParseMedia parses a media playlist into an ordered segment plan.
func ParseMedia(content string, baseURL *url.URL) (*Media, error) {
	if err := validateM3U8Header(content); err != nil {
		return nil, err
	}

	media := &Media{Live: true}
	var currentKey *Key
	var pendingByterange *ByteRange
	var pendingGap bool
	var pendingDuration float64
	var lastRangeEnd int64
	var sequence int64
	var initURI string

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line[0] != '#' {
			// Every non-comment line in a media playlist is a segment URI.
			seg := &Segment{
				URI:       resolveURL(baseURL, line),
				Sequence:  sequence,
				Byterange: pendingByterange,
				Key:       currentKey,
				Gap:       pendingGap,
				Duration:  pendingDuration,
			}
			if pendingByterange != nil {
				// Byterange segments have exact sizes straight from the playlist.
				seg.Size = pendingByterange.Length
			}
			sequence++
			pendingByterange = nil
			pendingGap = false
			pendingDuration = 0
			media.Segments = append(media.Segments, seg)
			continue
		}

		switch {
		case strings.HasPrefix(line, "#EXTINF:"):
			value := line[len("#EXTINF:"):]
			if idx := strings.Index(value, ","); idx >= 0 {
				value = value[:idx]
			}
			pendingDuration = parseFloat(value)
		case strings.HasPrefix(line, "#EXT-X-KEY:"):
			key, err := parseKey(line[len("#EXT-X-KEY:"):], baseURL)
			if err != nil {
				return nil, err
			}
			currentKey = key
		case strings.HasPrefix(line, "#EXT-X-MAP:"):
			attrs := parseAttributes(line[len("#EXT-X-MAP:"):])
			uriAttr := unquote(attrs["URI"])
			if uriAttr == "" {
				return nil, errors.New("EXT-X-MAP without URI is not supported")
			}
			resolved := resolveURL(baseURL, uriAttr)
			if initURI != "" && initURI != resolved {
				return nil, errors.New("multi-period fMP4 streams are not supported")
			}
			initURI = resolved
			if !media.IsFMP4 {
				// The init section is prepended to the segment plan, it takes
				// the media sequence number of the playlist for IV derivation.
				init := &Segment{
					URI:      resolved,
					Sequence: sequence,
					IsInit:   true,
					Key:      currentKey,
				}
				if attrs["BYTERANGE"] != "" {
					init.Byterange = parseByteRange(attrs["BYTERANGE"], 0)
					if init.Byterange == nil {
						return nil, fmt.Errorf("invalid EXT-X-MAP BYTERANGE: %s", attrs["BYTERANGE"])
					}
				}
				media.Segments = append(media.Segments, init)
			}
			media.IsFMP4 = true
		case strings.HasPrefix(line, "#EXT-X-BYTERANGE:"):
			pendingByterange = parseByteRange(line[len("#EXT-X-BYTERANGE:"):], lastRangeEnd)
			if pendingByterange == nil {
				return nil, fmt.Errorf("invalid EXT-X-BYTERANGE: %s", line)
			}
		case line == "#EXT-X-GAP":
			pendingGap = true
		case line == "#EXT-X-ENDLIST":
			media.Live = false
		case strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:"):
			sequence = parseInt64(line[len("#EXT-X-MEDIA-SEQUENCE:"):])
		case strings.HasPrefix(line, "#EXT-X-VERSION:"):
			media.Version = int(parseInt64(line[len("#EXT-X-VERSION:"):]))
		}

		if pendingByterange != nil {
			// Non-segment tags between BYTERANGE and its URI must not reset the
			// running offset, so it is tracked here after each parse.
			lastRangeEnd = pendingByterange.Offset + pendingByterange.Length
		}
	}

	if len(media.Segments) == 0 {
		return nil, errors.New("media playlist has no segments")
	}
	return media, nil
}

func parseKey(attrs string, baseURL *url.URL) (*Key, error) {
	parsed := parseAttributes(attrs)
	switch method := parsed["METHOD"]; method {
	case keyMethodNone:
		return nil, nil
	case keyMethodAES128:
		uri := unquote(parsed["URI"])
		if uri == "" {
			return nil, errors.New("AES-128 EXT-X-KEY without URI")
		}
		key := &Key{Method: keyMethodAES128, URI: resolveURL(baseURL, uri)}
		if iv, ok := parsed["IV"]; ok {
			value := strings.TrimPrefix(strings.ToLower(iv), "0x")
			decoded, err := hex.DecodeString(value)
			if err != nil || len(decoded) != 16 {
				return nil, fmt.Errorf("invalid EXT-X-KEY IV: %s", iv)
			}
			key.IV = decoded
		}
		return key, nil
	case keyMethodSampleAES:
		return nil, errors.New("SAMPLE-AES encryption is not supported")
	default:
		return nil, fmt.Errorf("unsupported encryption method: %s", method)
	}
}

func parseByteRange(value string, defaultOffset int64) *ByteRange {
	lengthPart := value
	offsetPart := ""
	if idx := strings.Index(value, "@"); idx >= 0 {
		lengthPart = value[:idx]
		offsetPart = value[idx+1:]
	}
	length := parseInt64(lengthPart)
	if length <= 0 {
		return nil
	}
	offset := defaultOffset
	if offsetPart != "" {
		offset = parseInt64(offsetPart)
		if offset < 0 {
			return nil
		}
	}
	return &ByteRange{Length: length, Offset: offset}
}

// resolveURL resolves a playlist URI reference against the playlist URL
// following RFC 3986.
func resolveURL(baseURL *url.URL, ref string) string {
	refURL, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return baseURL.ResolveReference(refURL).String()
}

// parseAttributes parses an HLS attribute list like FOO="bar",NUM=42.
func parseAttributes(s string) map[string]string {
	attrs := make(map[string]string)
	var key, val strings.Builder
	inKey := true
	inQuote := false
	flush := func() {
		name := strings.ToUpper(strings.TrimSpace(key.String()))
		if name != "" {
			attrs[name] = strings.TrimSpace(val.String())
		}
		key.Reset()
		val.Reset()
		inKey = true
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inKey {
			if c == '=' {
				inKey = false
			} else {
				key.WriteByte(c)
			}
			continue
		}
		if c == '"' {
			inQuote = !inQuote
			continue
		}
		if c == ',' && !inQuote {
			flush()
			continue
		}
		val.WriteByte(c)
	}
	flush()
	return attrs
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return s[1 : len(s)-1]
	}
	return s
}

func parseInt64(s string) int64 {
	value, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return value
}

func parseFloat(s string) float64 {
	value, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return value
}

// OutputExtension returns the container file extension for the playlist.
func (m *Media) OutputExtension() string {
	if m.IsFMP4 {
		return ".mp4"
	}
	return ".ts"
}

// DeriveOutputName builds the merged output file name from a playlist URL.
func DeriveOutputName(playlistURL string, media *Media) string {
	name := ""
	if u, err := url.Parse(playlistURL); err == nil {
		name = path.Base(u.Path)
		name = strings.TrimSuffix(name, path.Ext(name))
	}
	if name == "" || name == "/" || name == "." {
		if u, err := url.Parse(playlistURL); err == nil {
			name = u.Hostname()
		}
	}
	return name + media.OutputExtension()
}
