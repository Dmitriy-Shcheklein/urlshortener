package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditUrl(t *testing.T) {
	t.Run("Тест NewAuditUrl", func(t *testing.T) {
		t.Run("Должен создать AuditUrl", func(t *testing.T) {
			a := NewAuditUrl()

			assert.NotNil(t, a)
			assert.Empty(t, a.Host)
			assert.Zero(t, a.Port)
		})
	})

	t.Run("Тест Set", func(t *testing.T) {
		t.Run("Должен установить host и port", func(t *testing.T) {
			a := &AuditUrl{}
			err := a.Set("localhost:8080")

			require.NoError(t, err)
			assert.Equal(t, "localhost", a.Host)
			assert.Equal(t, 8080, a.Port)
		})

		t.Run("Должен вернуть ошибку при неверном формате", func(t *testing.T) {
			a := &AuditUrl{}

			err := a.Set("invalid")
			assert.Error(t, err)
		})

		t.Run("Должен вернуть ошибку при нечисловом порте", func(t *testing.T) {
			a := &AuditUrl{}

			err := a.Set("localhost:abc")
			assert.Error(t, err)
		})

		t.Run("Должен вернуть ошибку при пустой строке", func(t *testing.T) {
			a := &AuditUrl{}

			err := a.Set("")
			assert.Error(t, err)
		})

		t.Run("Должен корректно распарсить IP-адрес", func(t *testing.T) {
			a := &AuditUrl{}
			err := a.Set("127.0.0.1:9090")

			require.NoError(t, err)
			assert.Equal(t, "127.0.0.1", a.Host)
			assert.Equal(t, 9090, a.Port)
		})
	})

	t.Run("Тест String", func(t *testing.T) {
		t.Run("Должен вернуть host:port", func(t *testing.T) {
			a := &AuditUrl{Host: "example.com", Port: 3000}

			assert.Equal(t, "example.com:3000", a.String())
		})

		t.Run("Должен вернуть строку для пустого значения", func(t *testing.T) {
			a := &AuditUrl{}

			assert.Equal(t, ":0", a.String())
		})
	})
}
