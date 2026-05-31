package middlewares

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

type contextKey string

const userToken contextKey = "user_token"

var tokenExp = time.Hour

var errInvalidUserFormat = errors.New("invalid user format")

func (a *AppMiddleware) Auth(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("auth")
			if errors.Is(err, http.ErrNoCookie) {
				newCookie, gErr := a.generateNewCookie()
				if gErr != nil {
					a.logger.Error().Err(err).Msg("error while generate cookie")
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
				http.SetCookie(w, newCookie)
				claims, cErr := a.getClaims(newCookie)
				if cErr != nil {
					a.logger.Error().Err(err).Msg("error while getting claims")
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
				h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userToken, *claims)))
				return
			} else if err != nil {
				a.logger.Error().Err(err).Msg("error while getting cookie")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			withCtx, err := a.verifyToken(r, cookie)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			h.ServeHTTP(w, withCtx)
		},
	)
}

func (a *AppMiddleware) generateNewCookie() (*http.Cookie, error) {
	token, err := a.buildJWTString()
	if err != nil {
		return nil, err
	}
	return &http.Cookie{
		Name:     "auth",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86_400,
	}, nil
}

func (a *AppMiddleware) verifyToken(r *http.Request, cookie *http.Cookie) (*http.Request, error) {
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 3 {
		return r, fmt.Errorf("%w: invalid token format", errInvalidUserFormat)
	}

	claims := &Claims{}
	if _, err := jwt.ParseWithClaims(
		cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
			return a.cfg.GetSalt(), nil
		},
	); err != nil {
		return r, err
	}
	ctx := context.WithValue(r.Context(), userToken, *claims)
	return r.WithContext(ctx), nil
}

func (a *AppMiddleware) getClaims(cookie *http.Cookie) (*Claims, error) {
	claims := &Claims{}
	if _, err := jwt.ParseWithClaims(
		cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
			return a.cfg.GetSalt(), nil
		},
	); err != nil {
		return nil, err
	}
	return claims, nil
}

func (a *AppMiddleware) buildJWTString() (string, error) {
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256, Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
			},
			UserID: uuid.NewString(),
		},
	)

	tokenString, err := token.SignedString(a.cfg.GetSalt())
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func NewAuthService() *AuthService {
	return &AuthService{}
}

type AuthService struct{}

func (a *AuthService) GetUserID(ctx context.Context) ([]byte, error) {
	v, ok := ctx.Value(userToken).(Claims)
	if !ok {
		return nil, errors.New("error while getting UserID")
	}
	return []byte(v.UserID), nil
}
