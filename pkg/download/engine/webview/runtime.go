package webview

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
)

var ErrUnavailable = errors.New("gopeed runtime webview is unavailable")

type Opener interface {
	Open(opts OpenOptions) (Page, error)
}

type Page interface {
	AddInitScript(script string) error
	Navigate(url string, opts NavigateOptions) error
	Execute(expression string, args ...any) (any, error)
	GetCookies() ([]Cookie, error)
	SetCookie(cookie Cookie) error
	DeleteCookie(cookie Cookie) error
	ClearCookies() error
	Close() error
}

type OpenOptions struct {
	Headless  bool
	Debug     bool
	Title     string
	Width     int
	Height    int
	UserAgent string
}

type NavigateOptions struct {
	TimeoutMS int64
}

type WaitOptions struct {
	TimeoutMS      int64
	PollIntervalMS int64
}

type WaitForSelectorOptions struct {
	TimeoutMS      int64
	PollIntervalMS int64
	Visible        bool
	Hidden         bool
}

type Cookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain,omitempty"`
	Path     string    `json:"path,omitempty"`
	Expires  time.Time `json:"expires,omitempty"`
	Secure   bool      `json:"secure,omitempty"`
	HTTPOnly bool      `json:"httpOnly,omitempty"`
}

type Runtime struct {
	opener    Opener
	available bool

	mu      sync.Mutex
	pageSeq uint64
	pages   map[string]Page
}

func NewRuntime(opener Opener, available bool) *Runtime {
	if opener == nil {
		available = false
	}
	return &Runtime{
		opener:    opener,
		available: available,
		pages:     make(map[string]Page),
	}
}

func (r *Runtime) IsAvailable() bool {
	return r != nil && r.available && r.opener != nil
}

func (r *Runtime) Open(opts ...map[string]any) (*PageHandle, error) {
	if !r.IsAvailable() {
		return nil, ErrUnavailable
	}
	page, err := r.opener.Open(parseOpenOptions(firstMap(opts)))
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.pageSeq++
	id := fmt.Sprintf("page-%d", r.pageSeq)
	r.pages[id] = page
	return &PageHandle{runtime: r, id: id}, nil
}

func (r *Runtime) Close() error {
	r.mu.Lock()
	pageIDs := make([]string, 0, len(r.pages))
	for id := range r.pages {
		pageIDs = append(pageIDs, id)
	}
	r.mu.Unlock()

	var errs []error
	for _, id := range pageIDs {
		if err := r.closePage(id); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type PageHandle struct {
	runtime *Runtime
	id      string
}

func (p *PageHandle) AddInitScript(script string) error {
	page, err := p.page()
	if err != nil {
		return err
	}
	return page.AddInitScript(script)
}

func (p *PageHandle) Navigate(url string, opts ...map[string]any) error {
	page, err := p.page()
	if err != nil {
		return err
	}
	return page.Navigate(url, parseNavigateOptions(firstMap(opts)))
}

func (p *PageHandle) Execute(scriptOrFn any, args ...any) (any, error) {
	page, err := p.page()
	if err != nil {
		return nil, err
	}
	expression, err := normalizeExecutable(scriptOrFn)
	if err != nil {
		return nil, err
	}
	return page.Execute(expression, args...)
}

func (p *PageHandle) WaitForLoad(opts ...map[string]any) error {
	waitOpts := parseWaitOptions(firstMap(opts))
	_, matched, err := p.poll(waitOpts, func() (any, bool, error) {
		value, err := p.Execute(`() => document.readyState === "complete"`)
		if err != nil {
			return nil, false, err
		}
		return true, truthy(value), nil
	})
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("waitForLoad timeout")
	}
	return nil
}

func (p *PageHandle) WaitForSelector(selector string, opts ...map[string]any) (bool, error) {
	waitOpts := parseWaitForSelectorOptions(firstMap(opts))
	_, matched, err := p.poll(WaitOptions{
		TimeoutMS:      waitOpts.TimeoutMS,
		PollIntervalMS: waitOpts.PollIntervalMS,
	}, func() (any, bool, error) {
		value, err := p.Execute(`(selector, visible, hidden) => {
			const element = document.querySelector(selector);
			if (hidden) {
				if (!element) return true;
				const style = window.getComputedStyle(element);
				if (!style) return !!element.hidden;
				const rect = element.getBoundingClientRect();
				return element.hidden || style.display === "none" || style.visibility === "hidden" || rect.width === 0 || rect.height === 0;
			}
			if (!element) return false;
			if (!visible) return true;
			const style = window.getComputedStyle(element);
			if (!style) return true;
			const rect = element.getBoundingClientRect();
			return !element.hidden && style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0;
		}`, selector, waitOpts.Visible, waitOpts.Hidden)
		if err != nil {
			return nil, false, err
		}
		return true, truthy(value), nil
	})
	if err != nil {
		return false, err
	}
	return matched, nil
}

func (p *PageHandle) WaitForFunction(scriptOrFn any, args ...any) (any, error) {
	expression, err := normalizeExecutable(scriptOrFn)
	if err != nil {
		return nil, err
	}

	waitOpts := WaitOptions{}
	callArgs := args
	if len(args) > 0 {
		if raw, ok := args[0].(map[string]any); ok {
			waitOpts = parseWaitOptions(raw)
			callArgs = args[1:]
		}
	}

	value, matched, err := p.poll(waitOpts, func() (any, bool, error) {
		result, err := p.Execute(`(expression, args) => {
			const target = (0, eval)(expression);
			return Promise.resolve(typeof target === "function" ? target(...args) : target).then((value) => ({
				matched: !!value,
				value: value ?? null,
			}));
		}`, expression, callArgs)
		if err != nil {
			return nil, false, err
		}
		payload, ok := result.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("unexpected waitForFunction payload: %T", result)
		}
		return payload["value"], truthy(payload["matched"]), nil
	})
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, nil
	}
	return value, nil
}

func (p *PageHandle) GetCookies() ([]Cookie, error) {
	page, err := p.page()
	if err != nil {
		return nil, err
	}
	return page.GetCookies()
}

func (p *PageHandle) SetCookie(raw map[string]any) error {
	page, err := p.page()
	if err != nil {
		return err
	}
	return page.SetCookie(parseCookie(raw))
}

func (p *PageHandle) DeleteCookie(raw map[string]any) error {
	page, err := p.page()
	if err != nil {
		return err
	}
	return page.DeleteCookie(parseCookie(raw))
}

func (p *PageHandle) ClearCookies() error {
	page, err := p.page()
	if err != nil {
		return err
	}
	return page.ClearCookies()
}

func (p *PageHandle) URL() (string, error) {
	value, err := p.Execute(`() => location.href`)
	if err != nil {
		return "", err
	}
	return parseString(value), nil
}

func (p *PageHandle) Content() (string, error) {
	value, err := p.Execute(`() => document.documentElement ? document.documentElement.outerHTML : ""`)
	if err != nil {
		return "", err
	}
	return parseString(value), nil
}

func (p *PageHandle) Close() error {
	if p == nil || p.runtime == nil {
		return nil
	}
	return p.runtime.closePage(p.id)
}

func (p *PageHandle) page() (Page, error) {
	if p == nil || p.runtime == nil {
		return nil, fmt.Errorf("webview page handle is not initialized")
	}
	return p.runtime.getPage(p.id)
}

func (p *PageHandle) poll(opts WaitOptions, fn func() (any, bool, error)) (any, bool, error) {
	timeout := time.Duration(defaultWaitTimeoutMS(opts.TimeoutMS)) * time.Millisecond
	pollInterval := time.Duration(defaultPollIntervalMS(opts.PollIntervalMS)) * time.Millisecond
	deadline := time.Now().Add(timeout)
	for {
		value, matched, err := fn()
		if err != nil {
			return nil, false, err
		}
		if matched {
			return value, true, nil
		}
		if time.Now().After(deadline) {
			return nil, false, nil
		}
		time.Sleep(pollInterval)
	}
}

func (r *Runtime) getPage(id string) (Page, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	page, ok := r.pages[id]
	if !ok {
		return nil, fmt.Errorf("page not found")
	}
	return page, nil
}

func (r *Runtime) closePage(id string) error {
	r.mu.Lock()
	page, ok := r.pages[id]
	if !ok {
		r.mu.Unlock()
		return nil
	}
	delete(r.pages, id)
	r.mu.Unlock()
	return page.Close()
}

func parseOpenOptions(raw map[string]any) OpenOptions {
	return OpenOptions{
		Headless:  parseBool(raw["headless"]),
		Debug:     parseBool(raw["debug"]),
		Title:     parseString(raw["title"]),
		Width:     parseInt(raw["width"]),
		Height:    parseInt(raw["height"]),
		UserAgent: parseString(raw["userAgent"]),
	}
}

func parseNavigateOptions(raw map[string]any) NavigateOptions {
	return NavigateOptions{TimeoutMS: parseInt64(raw["timeoutMs"])}
}

func parseWaitOptions(raw map[string]any) WaitOptions {
	return WaitOptions{
		TimeoutMS:      parseInt64(raw["timeoutMs"]),
		PollIntervalMS: parseInt64(raw["pollIntervalMs"]),
	}
}

func parseWaitForSelectorOptions(raw map[string]any) WaitForSelectorOptions {
	return WaitForSelectorOptions{
		TimeoutMS:      parseInt64(raw["timeoutMs"]),
		PollIntervalMS: parseInt64(raw["pollIntervalMs"]),
		Visible:        parseBool(raw["visible"]),
		Hidden:         parseBool(raw["hidden"]),
	}
}

func parseCookie(raw map[string]any) Cookie {
	cookie := Cookie{
		Name:     parseString(raw["name"]),
		Value:    parseString(raw["value"]),
		Domain:   parseString(raw["domain"]),
		Path:     parseString(raw["path"]),
		Secure:   parseBool(raw["secure"]),
		HTTPOnly: parseBool(raw["httpOnly"]),
	}
	if rawExpires, ok := raw["expires"]; ok {
		cookie.Expires = parseTime(rawExpires)
	}
	return cookie
}

func defaultWaitTimeoutMS(value int64) int64 {
	if value <= 0 {
		return 10000
	}
	return value
}

func defaultPollIntervalMS(value int64) int64 {
	if value <= 0 {
		return 100
	}
	return value
}

func normalizeExecutable(scriptOrFn any) (string, error) {
	switch value := scriptOrFn.(type) {
	case string:
		return value, nil
	case *goja.Object:
		return NormalizeExecutableValue(value)
	default:
		return "", fmt.Errorf("execute expects a string or function, got %T", scriptOrFn)
	}
}

func NormalizeExecutableValue(value goja.Value) (string, error) {
	if value == nil {
		return "", fmt.Errorf("execute expects a string or function")
	}
	switch raw := value.Export().(type) {
	case string:
		return raw, nil
	}
	obj, ok := value.(*goja.Object)
	if ok {
		if _, ok := goja.AssertFunction(obj); ok {
			source, err := functionSource(obj)
			if err != nil {
				return "", err
			}
			return normalizeFunctionSource(source), nil
		}
	}
	source := strings.TrimSpace(value.String())
	if looksLikeFunctionSource(source) {
		return normalizeFunctionSource(source), nil
	}
	return "", fmt.Errorf("execute expects a string or function, got %T (%q)", value, source)
}

func normalizeFunctionSource(source string) string {
	repaired := repairArrowObjectLiteralSource(source)
	return "(" + repaired + ")"
}

func functionSource(fn *goja.Object) (string, error) {
	toStringValue := fn.Get("toString")
	toString, ok := goja.AssertFunction(toStringValue)
	if !ok {
		return "", fmt.Errorf("function.toString is not callable")
	}
	value, err := toString(fn)
	if err != nil {
		return "", err
	}
	return value.String(), nil
}

func repairArrowObjectLiteralSource(source string) string {
	trimmed := strings.TrimSpace(source)
	if !strings.Contains(trimmed, "=>") || !strings.Contains(trimmed, "({") {
		return source
	}
	if !strings.HasSuffix(trimmed, "}") {
		return source
	}
	if strings.Count(trimmed, "(")-strings.Count(trimmed, ")") != 1 {
		return source
	}
	if strings.Count(trimmed, "{") != strings.Count(trimmed, "}") {
		return source
	}
	// TODO: Remove this narrow repair once goja fixes toString() for concise arrow
	// functions that directly return object literals, then upstream the fix as a PR.
	return source + ")"
}

func looksLikeFunctionSource(source string) bool {
	if source == "" || source == "[object Object]" {
		return false
	}
	return strings.Contains(source, "=>") ||
		strings.HasPrefix(source, "function") ||
		strings.HasPrefix(source, "async function") ||
		strings.HasPrefix(source, "async (") ||
		strings.HasPrefix(source, "async(")
}

func truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	case int:
		return v != 0
	case int8:
		return v != 0
	case int16:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case uint:
		return v != 0
	case uint8:
		return v != 0
	case uint16:
		return v != 0
	case uint32:
		return v != 0
	case uint64:
		return v != 0
	case float32:
		return v != 0
	case float64:
		return v != 0
	default:
		return true
	}
}

func firstMap(values []map[string]any) map[string]any {
	if len(values) == 0 || values[0] == nil {
		return map[string]any{}
	}
	return values[0]
}

func parseString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func parseBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		parsed, _ := strconv.ParseBool(v)
		return parsed
	default:
		return false
	}
}

func parseInt(value any) int {
	return int(parseInt64(value))
}

func parseInt64(value any) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint:
		return int64(v)
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		parsed, _ := strconv.ParseInt(v, 10, 64)
		return parsed
	default:
		return 0
	}
}

func parseTime(value any) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v
	case int64:
		return time.UnixMilli(v)
	case int:
		return time.UnixMilli(int64(v))
	case float64:
		return time.UnixMilli(int64(v))
	case string:
		if parsed, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return parsed
		}
		if ms, err := strconv.ParseInt(v, 10, 64); err == nil {
			return time.UnixMilli(ms)
		}
	}
	return time.Time{}
}
