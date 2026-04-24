package rpcprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

type Provider struct {
	client *Client
}

func New(cfg enginewebview.RPCConfig) enginewebview.Provider {
	return &Provider{
		client: NewClient(cfg),
	}
}

func (p *Provider) IsAvailable() bool {
	if p == nil || p.client == nil || !p.client.Enabled() {
		return false
	}
	var result enginewebview.IsAvailableResult
	if err := p.client.Call(enginewebview.MethodIsAvailable, enginewebview.IsAvailableParams{}, &result); err != nil {
		return false
	}
	return result.Available
}

func (p *Provider) Open(opts enginewebview.OpenOptions) (enginewebview.Page, error) {
	if p == nil || p.client == nil || !p.IsAvailable() {
		return nil, enginewebview.ErrUnavailable
	}
	var result enginewebview.PageOpenResult
	if err := p.client.Call(enginewebview.MethodPageOpen, enginewebview.NewPageOpenParams(opts), &result); err != nil {
		return nil, err
	}
	return &page{
		client: p.client,
		id:     result.PageID,
	}, nil
}

type page struct {
	client *Client
	id     string
}

func (p *page) AddInitScript(script string) error {
	return p.client.Call(enginewebview.MethodPageAddInitScript, enginewebview.PageAddInitScriptParams{
		PageID: p.id,
		Script: script,
	}, nil)
}

func (p *page) Goto(url string, opts enginewebview.GotoOptions) error {
	return p.client.Call(enginewebview.MethodPageGoto, enginewebview.NewPageGotoParams(p.id, url, opts), nil)
}

func (p *page) Execute(expression string, args ...any) (any, error) {
	var result any
	if err := p.client.Call(enginewebview.MethodPageExecute, enginewebview.PageExecuteParams{
		PageID:     p.id,
		Expression: expression,
		Args:       args,
	}, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *page) GetCookies() ([]enginewebview.Cookie, error) {
	var result []enginewebview.Cookie
	if err := p.client.Call(enginewebview.MethodPageGetCookies, enginewebview.PageGetCookiesParams{
		PageID: p.id,
	}, &result); err != nil {
		return nil, err
	}
	if result == nil {
		return []enginewebview.Cookie{}, nil
	}
	return result, nil
}

func (p *page) SetCookie(cookie enginewebview.Cookie) error {
	return p.client.Call(enginewebview.MethodPageSetCookie, enginewebview.PageSetCookieParams{
		PageID: p.id,
		Cookie: cookie,
	}, nil)
}

func (p *page) DeleteCookie(cookie enginewebview.Cookie) error {
	return p.client.Call(enginewebview.MethodPageDeleteCookie, enginewebview.PageDeleteCookieParams{
		PageID: p.id,
		Cookie: cookie,
	}, nil)
}

func (p *page) ClearCookies() error {
	return p.client.Call(enginewebview.MethodPageClearCookies, enginewebview.PageClearCookiesParams{
		PageID: p.id,
	}, nil)
}

func (p *page) Close() error {
	return p.client.Call(enginewebview.MethodPageClose, enginewebview.PageCloseParams{
		PageID: p.id,
	}, nil)
}

type Client struct {
	endpoint string
	token    string
	http     *http.Client
}

func NewClient(cfg enginewebview.RPCConfig) *Client {
	client := &Client{token: cfg.Token}
	if !cfg.Enabled() {
		return client
	}
	switch strings.ToLower(cfg.Network) {
	case "unix":
		client.endpoint = "http://unix" + enginewebview.RPCEndpointPath
		client.http = &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", cfg.Address)
				},
			},
		}
	default:
		client.endpoint = "http://" + cfg.Address + enginewebview.RPCEndpointPath
		client.http = &http.Client{}
	}
	return client
}

func (c *Client) Enabled() bool {
	return c != nil && c.endpoint != "" && c.http != nil
}

func (c *Client) Call(method string, params any, result any) error {
	if !c.Enabled() {
		return enginewebview.ErrUnavailable
	}
	payload, err := json.Marshal(enginewebview.RPCRequest{
		Method: method,
		Params: params,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webview rpc status %d", resp.StatusCode)
	}

	var body enginewebview.RPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}
	if body.Error != nil {
		return body.Error
	}
	if result == nil {
		return nil
	}
	if len(body.Result) == 0 || string(body.Result) == "null" {
		return nil
	}
	return json.Unmarshal(body.Result, result)
}
