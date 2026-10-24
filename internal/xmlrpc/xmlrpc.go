// Package xmlrpc is a small XML-RPC client over HTTP(S) and SCGI, with the
// codec a test server needs. It has no dependencies beyond the standard library.
package xmlrpc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Client calls an XML-RPC endpoint over HTTP(S) or SCGI. rtorrent speaks
// XML-RPC over SCGI itself; a web server in front of it usually exposes the
// same calls over HTTP at /RPC2.
type Client struct {
	// endpoint is http://, https:// or scgi://host:port, or scgi:///path for a
	// Unix socket.
	endpoint *url.URL
	username string
	password string
	http     *http.Client
	timeout  time.Duration
}

// Fault is an error the server answered with.
type Fault struct {
	Code    int
	Message string
}

func (e *Fault) Error() string {
	return fmt.Sprintf("xml-rpc fault %d: %s", e.Code, e.Message)
}

// maxResponse bounds a response body. A status answer is a few
// kilobytes; a file list of a large torrent stays well below this.
const maxResponse = 32 << 20

func NewClient(rawURL, username, password string) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("address: %w", err)
	}
	switch u.Scheme {
	case "http", "https":
		if u.Host == "" {
			return nil, errors.New("address: missing host")
		}
	case "scgi":
		if u.Host == "" && u.Path == "" {
			return nil, errors.New("address: missing host or socket path")
		}
	default:
		return nil, fmt.Errorf("address: unsupported scheme %q, use http, https or scgi", u.Scheme)
	}
	return &Client{
		endpoint: u,
		username: username,
		password: password,
		http:     &http.Client{},
		timeout:  15 * time.Second,
	}, nil
}

// Call runs one method and returns its result: string, int64, bool, float64,
// []byte, []any or map[string]any. Parameters may be string, int, int64, bool,
// []byte, []any and map[string]any.
func (c *Client) Call(ctx context.Context, method string, params ...any) (any, error) {
	body, err := EncodeCall(method, params)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	var resp []byte
	if c.endpoint.Scheme == "scgi" {
		resp, err = c.postSCGI(ctx, body)
	} else {
		resp, err = c.postHTTP(ctx, body)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	v, err := DecodeResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	return v, nil
}

func (c *Client) postHTTP(ctx context.Context, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml")
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s", res.Status)
	}
	return readLimited(res.Body)
}

// postSCGI sends one SCGI request: a netstring of headers, then the body. The
// answer is CGI style, headers and a blank line, then the XML.
func (c *Client) postSCGI(ctx context.Context, body []byte) ([]byte, error) {
	network, address := "tcp", c.endpoint.Host
	if c.endpoint.Host == "" {
		network, address = "unix", c.endpoint.Path
	}
	var d net.Dialer
	conn, err := d.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	headers := "CONTENT_LENGTH\x00" + strconv.Itoa(len(body)) + "\x00SCGI\x001\x00"
	var req bytes.Buffer
	req.WriteString(strconv.Itoa(len(headers)))
	req.WriteByte(':')
	req.WriteString(headers)
	req.WriteByte(',')
	req.Write(body)
	if _, err := conn.Write(req.Bytes()); err != nil {
		return nil, err
	}
	r := bufio.NewReader(io.LimitReader(conn, maxResponse+1))
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("reading the SCGI answer: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if name, value, ok := strings.Cut(line, ":"); ok && strings.EqualFold(name, "Status") {
			if code := strings.Fields(value); len(code) > 0 && code[0] != "200" {
				return nil, fmt.Errorf("SCGI status %s", strings.TrimSpace(value))
			}
		}
	}
	return readLimited(r)
}

func readLimited(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, maxResponse+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxResponse {
		return nil, errors.New("answer too large")
	}
	return b, nil
}

func EncodeCall(method string, params []any) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0"?><methodCall><methodName>`)
	if err := xml.EscapeText(&b, []byte(method)); err != nil {
		return nil, err
	}
	b.WriteString(`</methodName><params>`)
	for _, p := range params {
		b.WriteString(`<param>`)
		if err := encodeValue(&b, p); err != nil {
			return nil, err
		}
		b.WriteString(`</param>`)
	}
	b.WriteString(`</params></methodCall>`)
	return b.Bytes(), nil
}

func encodeValue(b *bytes.Buffer, v any) error {
	b.WriteString(`<value>`)
	switch t := v.(type) {
	case string:
		b.WriteString(`<string>`)
		if err := xml.EscapeText(b, []byte(t)); err != nil {
			return err
		}
		b.WriteString(`</string>`)
	case int:
		encodeInt(b, int64(t))
	case int64:
		encodeInt(b, t)
	case bool:
		if t {
			b.WriteString(`<boolean>1</boolean>`)
		} else {
			b.WriteString(`<boolean>0</boolean>`)
		}
	case []byte:
		b.WriteString(`<base64>` + base64.StdEncoding.EncodeToString(t) + `</base64>`)
	case map[string]any:
		// Members go out in name order. rtorrent's system.multicall needs
		// methodName before params, which this order gives.
		names := make([]string, 0, len(t))
		for name := range t {
			names = append(names, name)
		}
		sort.Strings(names)
		b.WriteString(`<struct>`)
		for _, name := range names {
			e := t[name]
			b.WriteString(`<member><name>`)
			if err := xml.EscapeText(b, []byte(name)); err != nil {
				return err
			}
			b.WriteString(`</name>`)
			if err := encodeValue(b, e); err != nil {
				return err
			}
			b.WriteString(`</member>`)
		}
		b.WriteString(`</struct>`)
	case []any:
		b.WriteString(`<array><data>`)
		for _, e := range t {
			if err := encodeValue(b, e); err != nil {
				return err
			}
		}
		b.WriteString(`</data></array>`)
	default:
		return fmt.Errorf("xml-rpc: unsupported parameter type %T", v)
	}
	b.WriteString(`</value>`)
	return nil
}

// encodeInt writes <i4> when the value fits, which every server reads,
// and the <i8> extension otherwise.
func encodeInt(b *bytes.Buffer, n int64) {
	if n >= math.MinInt32 && n <= math.MaxInt32 {
		b.WriteString(`<i4>` + strconv.FormatInt(n, 10) + `</i4>`)
		return
	}
	b.WriteString(`<i8>` + strconv.FormatInt(n, 10) + `</i8>`)
}

// node is a generic element tree, enough for methodResponse.
type node struct {
	XMLName  xml.Name
	Content  string `xml:",chardata"`
	Children []node `xml:",any"`
}

func (n *node) child(name string) *node {
	for i := range n.Children {
		if n.Children[i].XMLName.Local == name {
			return &n.Children[i]
		}
	}
	return nil
}

func DecodeResponse(data []byte) (any, error) {
	var root node
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("unreadable answer: %w", err)
	}
	if root.XMLName.Local != "methodResponse" {
		return nil, fmt.Errorf("unexpected answer <%s>", root.XMLName.Local)
	}
	if fault := root.child("fault"); fault != nil {
		value := fault.child("value")
		if value == nil {
			return nil, &Fault{Message: "fault without a value"}
		}
		v, err := decodeValue(value)
		if err != nil {
			return nil, err
		}
		f := &Fault{}
		if m, ok := v.(map[string]any); ok {
			if code, ok := m["faultCode"].(int64); ok {
				f.Code = int(code)
			}
			f.Message, _ = m["faultString"].(string)
		}
		return nil, f
	}
	params := root.child("params")
	if params == nil {
		return nil, errors.New("answer without params")
	}
	param := params.child("param")
	if param == nil {
		return nil, nil
	}
	value := param.child("value")
	if value == nil {
		return nil, errors.New("param without a value")
	}
	return decodeValue(value)
}

func decodeValue(value *node) (any, error) {
	if len(value.Children) == 0 {
		// A value without a type element is a string.
		return value.Content, nil
	}
	t := &value.Children[0]
	switch t.XMLName.Local {
	case "string":
		return t.Content, nil
	case "i4", "i8", "int":
		n, err := strconv.ParseInt(strings.TrimSpace(t.Content), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bad integer %q", t.Content)
		}
		return n, nil
	case "boolean":
		return strings.TrimSpace(t.Content) == "1", nil
	case "double":
		f, err := strconv.ParseFloat(strings.TrimSpace(t.Content), 64)
		if err != nil {
			return nil, fmt.Errorf("bad double %q", t.Content)
		}
		return f, nil
	case "base64":
		return base64.StdEncoding.DecodeString(strings.TrimSpace(t.Content))
	case "array":
		data := t.child("data")
		out := []any{}
		if data == nil {
			return out, nil
		}
		for i := range data.Children {
			if data.Children[i].XMLName.Local != "value" {
				continue
			}
			v, err := decodeValue(&data.Children[i])
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case "struct":
		out := map[string]any{}
		for i := range t.Children {
			member := &t.Children[i]
			if member.XMLName.Local != "member" {
				continue
			}
			name, value := member.child("name"), member.child("value")
			if name == nil || value == nil {
				continue
			}
			v, err := decodeValue(value)
			if err != nil {
				return nil, err
			}
			out[name.Content] = v
		}
		return out, nil
	case "nil":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported value type <%s>", t.XMLName.Local)
	}
}

// DecodeCall reads a methodCall, for a test server.
func DecodeCall(data []byte) (string, []any, error) {
	var root node
	if err := xml.Unmarshal(data, &root); err != nil {
		return "", nil, err
	}
	if root.XMLName.Local != "methodCall" {
		return "", nil, fmt.Errorf("unexpected call <%s>", root.XMLName.Local)
	}
	name := root.child("methodName")
	if name == nil {
		return "", nil, errors.New("call without methodName")
	}
	var params []any
	if p := root.child("params"); p != nil {
		for i := range p.Children {
			value := p.Children[i].child("value")
			if value == nil {
				return "", nil, errors.New("param without a value")
			}
			v, err := decodeValue(value)
			if err != nil {
				return "", nil, err
			}
			params = append(params, v)
		}
	}
	return strings.TrimSpace(name.Content), params, nil
}

// EncodeResponse writes a methodResponse with one value, for a test server.
func EncodeResponse(v any) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0"?><methodResponse><params><param>`)
	if err := encodeValue(&b, v); err != nil {
		return nil, err
	}
	b.WriteString(`</param></params></methodResponse>`)
	return b.Bytes(), nil
}

// EncodeFault writes a fault response, for a test server.
func EncodeFault(code int, message string) []byte {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0"?><methodResponse><fault>`)
	_ = encodeValue(&b, map[string]any{"faultCode": code, "faultString": message})
	b.WriteString(`</fault></methodResponse>`)
	return b.Bytes()
}
