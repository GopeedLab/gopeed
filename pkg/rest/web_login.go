package rest

import (
	"crypto/subtle"
	_ "embed"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/GopeedLab/gopeed/pkg/rest/model"
)

const (
	webLoginPagePath = "/login"
	webLogoutPath    = "/logout"
)

// The web UI is rendered by Flutter on a canvas, so the login screen has no DOM
// inputs and browser password managers cannot fill it in. When enabled, this
// plain HTML page is served before the web UI is loaded and reuses the same
// cookie session as the API login endpoint.
//
//go:embed web_login.html
var webLoginHTML string

var webLoginTemplate = template.Must(template.New("web-login").Parse(webLoginHTML))

type webLoginPageData struct {
	Lang          string
	Title         string
	UsernameLabel string
	PasswordLabel string
	LoginButton   string
	Hint          string
	Error         string
	Username      string
	Next          string
}

type webLoginPageHandler struct {
	auth     *model.WebAuth
	sessions *webSessionStore
}

func newWebLoginPageHandler(auth *model.WebAuth, sessions *webSessionStore) *webLoginPageHandler {
	return &webLoginPageHandler{auth: auth, sessions: sessions}
}

func (h *webLoginPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.isAuthenticated(r) {
		http.Redirect(w, r, safeNextPath(r.URL.Query().Get("next")), http.StatusSeeOther)
		return
	}
	h.render(w, r, http.StatusOK, "", "", safeNextPath(r.URL.Query().Get("next")))
}

func (h *webLoginPageHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, http.StatusBadRequest, "", "", "/")
		return
	}
	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")
	next := safeNextPath(r.PostForm.Get("next"))
	if !validWebCredentials(h.auth, username, password) {
		h.render(w, r, http.StatusUnauthorized, loginPageText(r).invalidCredentials, username, next)
		return
	}
	sessionID, expiresAt, err := h.sessions.create()
	if err != nil {
		h.render(w, r, http.StatusInternalServerError, loginPageText(r).serverError, username, next)
		return
	}
	setWebSessionCookie(w, r, sessionID, expiresAt)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (h *webLoginPageHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	clearWebSessionCookie(w)
	http.Redirect(w, r, webLoginPagePath, http.StatusSeeOther)
}

func (h *webLoginPageHandler) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie(webAuthCookieName)
	return err == nil && h.sessions.valid(cookie.Value)
}

func (h *webLoginPageHandler) render(w http.ResponseWriter, r *http.Request, status int, message string, username string, next string) {
	text := loginPageText(r)
	data := webLoginPageData{
		Lang:          text.lang,
		Title:         text.title,
		UsernameLabel: text.username,
		PasswordLabel: text.password,
		LoginButton:   text.login,
		Hint:          text.hint,
		Error:         message,
		Username:      username,
		Next:          next,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = webLoginTemplate.Execute(w, data)
}

type webLoginText struct {
	lang, title, username, password, login, hint, invalidCredentials, serverError string
}

func loginPageText(r *http.Request) webLoginText {
	// Only the highest-weighted language tag decides the copy.
	primary := strings.TrimSpace(strings.Split(r.Header.Get("Accept-Language"), ",")[0])
	if strings.HasPrefix(strings.ToLower(primary), "zh") {
		return webLoginText{
			lang:               "zh-CN",
			title:              "登录 - Gopeed",
			username:           "用户名",
			password:           "密码",
			login:              "登录",
			hint:               "登录状态保持 7 天",
			invalidCredentials: "用户名或密码错误",
			serverError:        "登录失败，请查看服务端日志",
		}
	}
	return webLoginText{
		lang:               "en",
		title:              "Sign in - Gopeed",
		username:           "Username",
		password:           "Password",
		login:              "Sign in",
		hint:               "You stay signed in for 7 days",
		invalidCredentials: "Incorrect username or password",
		serverError:        "Sign-in failed, check the server logs",
	}
}

func validWebCredentials(auth *model.WebAuth, username, password string) bool {
	if auth == nil {
		return false
	}
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(auth.Username)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(auth.Password)) == 1
	return userOK && passOK
}

func setWebSessionCookie(w http.ResponseWriter, r *http.Request, sessionID string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     webAuthCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(webAuthSessionTTL.Seconds()),
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   requestUsesHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func clearWebSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     webAuthCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// isHTMLNavigation reports whether the request is a top level browser
// navigation, which is the only case that should be redirected to the login
// page. Sub resources (scripts, images, API calls) keep their normal handling.
func isHTMLNavigation(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	return strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/html")
}

// safeNextPath only allows redirects to local absolute paths, which prevents
// the login page from being used as an open redirect.
func safeNextPath(next string) string {
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
		return "/"
	}
	if _, err := url.Parse(next); err != nil {
		return "/"
	}
	return next
}
