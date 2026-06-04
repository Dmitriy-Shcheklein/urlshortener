package fsobserver

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLogger() *zerolog.Logger {
	return new(zerolog.Nop())
}

func TestObserver(t *testing.T) {
	t.Run(
		"Тест создания Observer", func(t *testing.T) {
			t.Run(
				"Должен создать Observer", func(t *testing.T) {
					logger := newLogger()
					o := New(logger, "/tmp/audit.log")

					assert.NotNil(t, o)
					assert.Equal(t, "/tmp/audit.log", o.path)
					assert.Equal(t, logger, o.logger)
				},
			)
		},
	)

	t.Run(
		"Тест HandleMessage", func(t *testing.T) {
			t.Run(
				"Должен записать сообщение в файл", func(t *testing.T) {
					dir := t.TempDir()
					filePath := filepath.Join(dir, "audit.log")

					o := New(newLogger(), filePath)

					msg := model.AuditMsg{
						Ts: 1234567890, Action: "create", UserID: new("user1"), URL: "http://example.com",
					}
					o.HandleMessage(msg)

					// #nosec G304
					file, err := os.Open(filePath)
					require.NoError(t, err)
					defer func(file *os.File) {
						_ = file.Close()
					}(file)

					scanner := bufio.NewScanner(file)
					require.True(t, scanner.Scan())

					var got model.AuditMsg
					err = json.Unmarshal(scanner.Bytes(), &got)
					require.NoError(t, err)

					assert.Equal(t, msg.Ts, got.Ts)
					assert.Equal(t, msg.Action, got.Action)
					assert.Equal(t, msg.URL, got.URL)
					assert.Equal(t, msg.UserID, got.UserID)
				},
			)

			t.Run(
				"Должен дописать второе сообщение в файл", func(t *testing.T) {
					dir := t.TempDir()
					filePath := filepath.Join(dir, "audit.log")

					o := New(newLogger(), filePath)

					msg1 := model.AuditMsg{Ts: 1, Action: "create", URL: "http://first.com"}
					msg2 := model.AuditMsg{Ts: 2, Action: "delete", URL: "http://second.com"}

					o.HandleMessage(msg1)
					o.HandleMessage(msg2)

					// #nosec G304
					file, err := os.Open(filePath)
					require.NoError(t, err)
					defer func(file *os.File) {
						_ = file.Close()
					}(file)

					scanner := bufio.NewScanner(file)
					lines := 0
					for scanner.Scan() {
						lines++
					}
					assert.Equal(t, 2, lines)
				},
			)

			t.Run(
				"Должен не паниковать при ошибке открытия файла", func(t *testing.T) {
					o := New(newLogger(), "/nonexistent/dir/audit.log")

					msg := model.AuditMsg{Ts: 1, Action: "create", URL: "http://example.com"}

					assert.NotPanics(
						t, func() {
							o.HandleMessage(msg)
						},
					)
				},
			)

			t.Run(
				"Должен записать сообщение без UserID", func(t *testing.T) {
					dir := t.TempDir()
					filePath := filepath.Join(dir, "audit.log")

					o := New(newLogger(), filePath)

					msg := model.AuditMsg{Ts: 100, Action: "create", URL: "http://example.com"}
					o.HandleMessage(msg)

					// #nosec G304
					file, err := os.Open(filePath)
					require.NoError(t, err)
					defer func(file *os.File) {
						_ = file.Close()
					}(file)

					scanner := bufio.NewScanner(file)
					require.True(t, scanner.Scan())

					var got model.AuditMsg
					err = json.Unmarshal(scanner.Bytes(), &got)
					require.NoError(t, err)

					assert.Nil(t, got.UserID)
					assert.Equal(t, "create", got.Action)
					assert.Equal(t, "http://example.com", got.URL)
				},
			)
		},
	)
}
