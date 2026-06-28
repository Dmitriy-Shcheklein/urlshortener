package healthcheck_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/handler/healthcheck"
)

// mockService implements healthcheck.Service for examples.
type mockService struct {
	healthy bool
}

func (m *mockService) PingDB() error {
	if !m.healthy {
		return fmt.Errorf("database unavailable")
	}
	return nil
}

func ExampleHandler_PingDB() {
	svc := &mockService{healthy: true}
	handler, _ := healthcheck.New(svc)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	handler.PingDB(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)

	// Output:
	// Status: 200
}

func ExampleHandler_PingDB_unhealthy() {
	svc := &mockService{healthy: false}
	handler, _ := healthcheck.New(svc)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	handler.PingDB(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)

	// Output:
	// Status: 500
}
