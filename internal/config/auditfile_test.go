package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditFilePath(t *testing.T) {
	t.Run("Тест NewAuditFilePath", func(t *testing.T) {
		t.Run("Должен создать AuditFilePath с дефолтным путём", func(t *testing.T) {
			f := NewAuditFilePath()

			assert.NotNil(t, f)
			assert.Equal(t, "", f.Path)
			assert.False(t, f.IsFromEnv)
		})
	})

	t.Run("Тест Set", func(t *testing.T) {
		t.Run("Должен установить путь", func(t *testing.T) {
			f := &AuditFilePath{}

			err := f.Set("/var/log/audit.log")

			require.NoError(t, err)
			assert.Equal(t, "/var/log/audit.log", f.Path)
		})

		t.Run("Должен не менять путь если IsFromEnv=true", func(t *testing.T) {
			f := &AuditFilePath{Path: "/original", IsFromEnv: true}

			err := f.Set("/new/path")

			require.NoError(t, err)
			assert.Equal(t, "/original", f.Path)
		})

		t.Run("Должен установить путь если IsFromEnv=false", func(t *testing.T) {
			f := &AuditFilePath{Path: "/original", IsFromEnv: false}

			err := f.Set("/new/path")

			require.NoError(t, err)
			assert.Equal(t, "/new/path", f.Path)
		})
	})

	t.Run("Тест String", func(t *testing.T) {
		t.Run("Должен вернуть текущий путь", func(t *testing.T) {
			f := &AuditFilePath{Path: "/tmp/audit.log"}

			assert.Equal(t, "/tmp/audit.log", f.String())
		})

		t.Run("Должен вернуть пустую строку для пустого пути", func(t *testing.T) {
			f := &AuditFilePath{}

			assert.Equal(t, "", f.String())
		})
	})
}
