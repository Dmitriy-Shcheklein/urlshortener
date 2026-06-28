package shortener_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/handler/shortener"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog"
)

// mockService implements shortener.Service for examples.
type mockService struct {
	urls map[string][]byte
}

func (m *mockService) GetByID(id string) ([]byte, error) {
	if url, ok := m.urls[id]; ok {
		return url, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockService) CreateShort(originalURL []byte, userID []byte) ([]byte, error) {
	short := fmt.Sprintf("abc%d", len(m.urls))
	m.urls[short] = originalURL
	return []byte(short), nil
}

func (m *mockService) CreateMany(values []model.CreateManyBodyRaw, userID []byte) ([]model.CreateManyResponseRaw, error) {
	result := make([]model.CreateManyResponseRaw, len(values))
	for i, v := range values {
		short := fmt.Sprintf("batch%d", i)
		m.urls[short] = []byte(v.OriginalURL)
		result[i] = model.CreateManyResponseRaw{
			CorrelationID: v.CorrelationID,
			ShortURL:      short,
		}
	}
	return result, nil
}

func (m *mockService) FindByUserID(userID []byte) ([]model.LinkRow, error) {
	var rows []model.LinkRow
	for short, original := range m.urls {
		rows = append(rows, model.LinkRow{
			ShortURL:    short,
			OriginalURL: string(original),
		})
	}
	return rows, nil
}

// mockConfig implements shortener.Config for examples.
type mockConfig struct {
	baseAddress []byte
}

func (m *mockConfig) GetBaseAddress() []byte {
	return m.baseAddress
}

// mockDeleteWorker implements shortener.DeleteWorker for examples.
type mockDeleteWorker struct{}

func (m *mockDeleteWorker) AddToQueue(urls []string, userID string) {}

// mockAuthService implements shortener.AuthService for examples.
type mockAuthService struct{}

func (m *mockAuthService) GetUserID(ctx context.Context) ([]byte, error) {
	return []byte("user123"), nil
}

// mockAuditor implements shortener.Auditor for examples.
type mockAuditor struct{}

func (m *mockAuditor) Audit(userID *string, action string, URL string) {}

func setupHandler() *shortener.Handler {
	nop := zerolog.Nop()
	logger.Logger = &nop
	svc := &mockService{urls: make(map[string][]byte)}
	cfg := &mockConfig{baseAddress: []byte("http://localhost:8080")}
	handler, _ := shortener.New(svc, cfg, &mockDeleteWorker{}, &mockAuthService{}, &mockAuditor{})
	return handler
}

func ExampleHandler_CreateShort() {
	handler := setupHandler()

	body := bytes.NewBufferString("https://practicum.yandex.ru")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler.CreateShort(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Content-Type:", resp.Header.Get("Content-Type"))

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println("Body:", string(respBody))

	// Output:
	// Status: 201
	// Content-Type: text/plain
	// Body: http://localhost:8080/abc0
}

func ExampleHandler_CreateFromJSONBody() {
	handler := setupHandler()

	requestBody := `{"url": "https://practicum.yandex.ru"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateFromJSONBody(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)

	var result shortener.CreateShortResponse
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Println("Has result:", result.Result != "")

	// Output:
	// Status: 201
	// Has result: true
}

func ExampleHandler_CreateMany() {
	handler := setupHandler()

	requestBody := `[{"correlation_id": "1", "original_url": "https://practicum.yandex.ru"}]`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateMany(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)

	var result []model.CreateManyResponseRaw
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Println("Items:", len(result))
	fmt.Println("CorrelationID:", result[0].CorrelationID)

	// Output:
	// Status: 201
	// Items: 1
	// CorrelationID: 1
}

func ExampleHandler_GetByID() {
	handler := setupHandler()

	// First create a short URL via JSON API
	requestBody := `{"url": "https://practicum.yandex.ru"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(requestBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	handler.CreateFromJSONBody(createW, createReq)

	// Parse the created short URL to get the ID
	var createResp shortener.CreateShortResponse
	json.NewDecoder(createW.Result().Body).Decode(&createResp)

	// Extract ID from the short URL (remove base address prefix)
	id := createResp.Result[len("http://localhost:8080/"):]

	req := httptest.NewRequest(http.MethodGet, "/"+id, nil)
	w := httptest.NewRecorder()

	handler.GetByID(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Location:", resp.Header.Get("Location"))

	// Output:
	// Status: 307
	// Location: https://practicum.yandex.ru
}

func ExampleHandler_GetByUserID() {
	handler := setupHandler()

	// First create a short URL
	body := bytes.NewBufferString("https://practicum.yandex.ru")
	createReq := httptest.NewRequest(http.MethodPost, "/", body)
	createReq.Header.Set("Content-Type", "text/plain")
	createW := httptest.NewRecorder()
	handler.CreateShort(createW, createReq)

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	w := httptest.NewRecorder()

	handler.GetByUserID(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Content-Type:", resp.Header.Get("Content-Type"))

	var result []map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Println("URLs count:", len(result))

	// Output:
	// Status: 200
	// Content-Type: application/json
	// URLs count: 1
}

func ExampleHandler_DeleteLinks() {
	handler := setupHandler()

	requestBody := `["abc123", "def456"]`
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.DeleteLinks(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)

	// Output:
	// Status: 202
}
