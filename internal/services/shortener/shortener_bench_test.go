package shortener

import (
	"fmt"
	"testing"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/stretchr/testify/mock"
)

func BenchmarkShortenURLCRC32(b *testing.B) {
	url := []byte("https://example.com/very/long/url/to/shorten")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		shortenURLCRC32(url)
	}
}

func BenchmarkCreateShort(b *testing.B) {
	repo := NewMockLinkRepository(b)
	repo.EXPECT().Save(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	svc := New(repo)
	userID := []byte("test-user-id")
	url := []byte("https://example.com/some/long/path")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.CreateShort(url, userID)
	}
}

func BenchmarkCreateMany(b *testing.B) {
	for _, count := range []int{1, 10, 100} {
		b.Run(
			fmt.Sprintf("count_%d", count), func(b *testing.B) {
				repo := NewMockLinkRepository(b)
				repo.EXPECT().SaveMany(mock.Anything, mock.Anything).Return(nil)

				svc := New(repo)
				userID := []byte("test-user-id")

				values := make([]model.CreateManyBodyRaw, count)
				for i := 0; i < count; i++ {
					values[i] = model.CreateManyBodyRaw{
						CorrelationID: fmt.Sprintf("corr-id-%d", i),
						OriginalURL:   fmt.Sprintf("https://example.com/path/%d", i),
					}
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = svc.CreateMany(values, userID)
				}
			},
		)
	}
}

func BenchmarkGetByID(b *testing.B) {
	repo := NewMockLinkRepository(b)
	repo.EXPECT().GetByID(mock.Anything).Return([]byte("https://example.com"), nil)

	svc := New(repo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.GetByID("abc123")
	}
}

func BenchmarkFindByUserID(b *testing.B) {
	repo := NewMockLinkRepository(b)
	rows := []model.LinkRow{
		{ID: "1", ShortURL: "abc", OriginalURL: "https://example.com/1", UserID: "user"},
		{ID: "2", ShortURL: "def", OriginalURL: "https://example.com/2", UserID: "user"},
	}
	repo.EXPECT().FindByUserID(mock.Anything).Return(rows, nil)

	svc := New(repo)
	userID := []byte("test-user-id")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.FindByUserID(userID)
	}
}
