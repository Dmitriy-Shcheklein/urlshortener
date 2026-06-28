package shortener_test

import (
	"fmt"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/services/shortener"
)

// mockRepository implements shortener.LinkRepository for examples.
type mockRepository struct {
	urls map[string][]byte
}

func (m *mockRepository) GetByID(id string) ([]byte, error) {
	if url, ok := m.urls[id]; ok {
		return url, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockRepository) Save(path []byte, short []byte, userID []byte) error {
	m.urls[string(short)] = path
	return nil
}

func (m *mockRepository) SaveMany(values []model.LinkRow, userID []byte) error {
	for _, v := range values {
		m.urls[v.ShortURL] = []byte(v.OriginalURL)
	}
	return nil
}

func (m *mockRepository) FindByUserID(userID []byte) ([]model.LinkRow, error) {
	var rows []model.LinkRow
	for short, original := range m.urls {
		rows = append(rows, model.LinkRow{
			ShortURL:    short,
			OriginalURL: string(original),
		})
	}
	return rows, nil
}

func (m *mockRepository) Delete(in []*model.LinkToDelete) error {
	for _, item := range in {
		delete(m.urls, item.Link)
	}
	return nil
}

func ExampleNew() {
	repo := &mockRepository{urls: make(map[string][]byte)}
	svc := shortener.New(repo)

	short, err := svc.CreateShort([]byte("https://practicum.yandex.ru"), []byte("user1"))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Short URL created:", len(short) > 0)

	// Output:
	// Short URL created: true
}

func ExampleService_CreateShort() {
	repo := &mockRepository{urls: make(map[string][]byte)}
	svc := shortener.New(repo)

	short, err := svc.CreateShort([]byte("https://practicum.yandex.ru"), []byte("user1"))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	original, err := svc.GetByID(string(short))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Original URL:", string(original))

	// Output:
	// Original URL: https://practicum.yandex.ru
}

func ExampleService_CreateMany() {
	repo := &mockRepository{urls: make(map[string][]byte)}
	svc := shortener.New(repo)

	items := []model.CreateManyBodyRaw{
		{CorrelationID: "1", OriginalURL: "https://practicum.yandex.ru"},
		{CorrelationID: "2", OriginalURL: "https://ya.ru"},
	}

	result, err := svc.CreateMany(items, []byte("user1"))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Items created:", len(result))
	fmt.Println("First correlation:", result[0].CorrelationID)

	// Output:
	// Items created: 2
	// First correlation: 1
}

func ExampleService_FindByUserID() {
	repo := &mockRepository{urls: make(map[string][]byte)}
	svc := shortener.New(repo)

	svc.CreateShort([]byte("https://practicum.yandex.ru"), []byte("user1"))
	svc.CreateShort([]byte("https://ya.ru"), []byte("user1"))

	urls, err := svc.FindByUserID([]byte("user1"))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("URLs found:", len(urls))

	// Output:
	// URLs found: 2
}

func ExampleService_Delete() {
	repo := &mockRepository{urls: make(map[string][]byte)}
	svc := shortener.New(repo)

	short, _ := svc.CreateShort([]byte("https://practicum.yandex.ru"), []byte("user1"))

	err := svc.Delete([]*model.LinkToDelete{
		{Link: string(short), UserID: "user1"},
	})

	fmt.Println("Delete error:", err)

	// Output:
	// Delete error: <nil>
}
