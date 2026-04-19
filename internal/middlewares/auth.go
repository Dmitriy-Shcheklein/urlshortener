package middlewares

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

var secretKey = []byte("secret_key")

type contextKey string

const UserIDKey contextKey = "user_id"

type InvalidUserFormatError struct {
	message string
}

func (i *InvalidUserFormatError) Error() string {
	return i.message
}

func Auth(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("auth")
			if errors.Is(err, http.ErrNoCookie) {
				newCookie, userID := generateNewCookie()
				http.SetCookie(w, newCookie)
				h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), UserIDKey, userID)))
				return
			} else if err != nil {
				http.Error(w, "error while getting cookie", http.StatusInternalServerError)
				return
			}

			withCtx, err := verifyToken(r, cookie)
			if _, ok := errors.AsType[*InvalidUserFormatError](err); ok {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			if err != nil {
				http.Error(w, fmt.Sprintf("error while verify cookie: %v", err), http.StatusInternalServerError)
				return
			}
			h.ServeHTTP(w, withCtx)
		},
	)
}

func generateNewCookie() (*http.Cookie, []byte) {
	userID := uuid.NewString()
	userIDBytes := []byte(userID)

	encodedUserID := base64.URLEncoding.EncodeToString(userIDBytes)

	h := hmac.New(sha256.New, secretKey)
	h.Write(userIDBytes)
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	return &http.Cookie{
		Name:     "auth",
		Value:    encodedUserID + "." + signature,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86_400,
	}, userIDBytes
}

func verifyToken(r *http.Request, cookie *http.Cookie) (*http.Request, error) {
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return r, &InvalidUserFormatError{message: "invalid cookie format"}
	}
	encodedUserID := parts[0]
	receivedSignature := parts[1]

	userIDBytes, err := base64.URLEncoding.DecodeString(encodedUserID)
	if err != nil {
		return r, &InvalidUserFormatError{message: "invalid encoded userID"}
	}

	h := hmac.New(sha256.New, secretKey)
	h.Write(userIDBytes)

	expectedSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	if !hmac.Equal([]byte(receivedSignature), []byte(expectedSignature)) {
		return r, &InvalidUserFormatError{message: "invalid signature"}
	}

	ctx := context.WithValue(r.Context(), UserIDKey, userIDBytes)
	return r.WithContext(ctx), nil
}
