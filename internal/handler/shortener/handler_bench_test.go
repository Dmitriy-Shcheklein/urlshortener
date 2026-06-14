package shortener

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
)

func BenchmarkCreateShortHandler(b *testing.B) {
	svc := NewMockService(b)
	svc.EXPECT().CreateShort(mock.Anything, mock.Anything).Return([]byte("abc123"), nil)

	cfg := NewMockConfig(b)
	cfg.EXPECT().GetBaseAddress().Return([]byte("http://localhost:8080"))

	dw := NewMockDeleteWorker(b)
	auth := NewMockAuthService(b)
	auth.EXPECT().GetUserID(mock.Anything).Return([]byte("test-user"), nil)

	auditor := NewMockAuditor(b)
	auditor.EXPECT().Audit(mock.Anything, mock.Anything, mock.Anything)

	handler, err := New(svc, cfg, dw, auth, auditor)
	if err != nil {
		b.Fatal(err)
	}

	body := bytes.NewBufferString("https://example.com/long/path/to/shorten")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		handler.CreateShort(w, req)
	}
}

func BenchmarkCreateFromJSONBody(b *testing.B) {
	svc := NewMockService(b)
	svc.EXPECT().CreateShort(mock.Anything, mock.Anything).Return([]byte("abc123"), nil)

	cfg := NewMockConfig(b)
	cfg.EXPECT().GetBaseAddress().Return([]byte("http://localhost:8080"))

	dw := NewMockDeleteWorker(b)
	auth := NewMockAuthService(b)
	auth.EXPECT().GetUserID(mock.Anything).Return([]byte("test-user"), nil)

	auditor := NewMockAuditor(b)
	auditor.EXPECT().Audit(mock.Anything, mock.Anything, mock.Anything)

	handler, err := New(svc, cfg, dw, auth, auditor)
	if err != nil {
		b.Fatal(err)
	}

	jsonBody := `{"url":"https://example.com/long/path/to/shorten"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateFromJSONBody(w, req)
	}
}
