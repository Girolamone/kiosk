package auth

import (
	"context"
	"net/http"
)

const sessionCookieName = "kiosk_session"

// SessionWriter lets a resolver open or close a session without knowing that
// a session happens to be an HTTP cookie.
type SessionWriter interface {
	Start(token string)
	End()
}

type sessionWriterKey struct{}

// SessionWriterFromContext returns the session writer for the current request.
func SessionWriterFromContext(ctx context.Context) (SessionWriter, bool) {
	sw, ok := ctx.Value(sessionWriterKey{}).(SessionWriter)
	return sw, ok
}

// Middleware reads the session cookie and, when it is valid, attaches the
// caller to the request context. An absent or bad cookie is not an error:
// the request simply continues as anonymous, and whatever it tries to reach
// decides whether that is allowed.
func Middleware(issuer *TokenIssuer, secureCookies bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), sessionWriterKey{}, &cookieSession{
				w:      w,
				issuer: issuer,
				secure: secureCookies,
			})

			if cookie, err := r.Cookie(sessionCookieName); err == nil {
				if user, err := issuer.Parse(cookie.Value); err == nil {
					ctx = NewContext(ctx, user)
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type cookieSession struct {
	w      http.ResponseWriter
	issuer *TokenIssuer
	secure bool
}

func (c *cookieSession) Start(token string) {
	http.SetCookie(c.w, c.cookie(token, int(c.issuer.TTL().Seconds())))
}

func (c *cookieSession) End() {
	// A negative MaxAge tells the browser to delete the cookie now.
	http.SetCookie(c.w, c.cookie("", -1))
}

func (c *cookieSession) cookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:  sessionCookieName,
		Value: value,
		Path:  "/",
		// HttpOnly keeps the token out of reach of JavaScript, so a cross-site
		// scripting bug cannot walk off with the session.
		HttpOnly: true,
		Secure:   c.secure,
		// Lax works because the API and the web app are served from one
		// origin. Splitting them across domains would need SameSiteNoneMode
		// and Secure, plus CORS configured to allow credentials.
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
}
