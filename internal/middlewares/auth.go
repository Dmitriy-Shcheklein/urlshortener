package middlewares

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

var secretKey = []byte("secret_key")

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

type contextKey string

const userToken contextKey = "user_token"

var tokenExp = time.Hour

var errInvalidUserFormat = errors.New("invalid user format")

func Auth(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("auth")
			if errors.Is(err, http.ErrNoCookie) {
				newCookie, gErr := generateNewCookie()
				if gErr != nil {
					logger.Logger.Error().Err(err).Msg("error while generate cookie")
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
				http.SetCookie(w, newCookie)
				claims, cErr := getClaims(newCookie)
				if cErr != nil {
					logger.Logger.Error().Err(err).Msg("error while getting claims")
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
				h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userToken, *claims)))
				return
			} else if err != nil {
				logger.Logger.Error().Err(err).Msg("error while getting cookie")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			withCtx, err := verifyToken(r, cookie)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			h.ServeHTTP(w, withCtx)
		},
	)
}

func generateNewCookie() (*http.Cookie, error) {
	token, err := buildJWTString()
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

func verifyToken(r *http.Request, cookie *http.Cookie) (*http.Request, error) {
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 3 {
		return r, fmt.Errorf("%w: invalid token format", errInvalidUserFormat)
	}

	claims := &Claims{}
	if _, err := jwt.ParseWithClaims(
		cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
			return secretKey, nil
		},
	); err != nil {
		return r, err
	}
	ctx := context.WithValue(r.Context(), userToken, *claims)
	return r.WithContext(ctx), nil
}

func getClaims(cookie *http.Cookie) (*Claims, error) {
	claims := &Claims{}
	if _, err := jwt.ParseWithClaims(
		cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
			return secretKey, nil
		},
	); err != nil {
		return nil, err
	}
	return claims, nil
}

func buildJWTString() (string, error) {
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256, Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
			},
			UserID: uuid.NewString(),
		},
	)

	tokenString, err := token.SignedString(secretKey)
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
