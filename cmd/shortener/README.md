# cmd/shortener

В данной директории содержится код, который скомпилируется в бинарное приложение.

Рекомендуется помещать только код, необходимый для запуска приложения, но не бизнес-логику.

Название директории должно соответствовать названию приложения.

Директория `cmd/shortener` содержит:
- точку входа в приложение (функция `main`)
- инициализацию зависимостей (можно вынести в отдельный пакет `internal/app`)
- настройку и запуск HTTP-сервера (можно вынести в отдельный пакет `internal/router`)
- обработку сигналов завершения работы приложения

## Сборка с использованием -ldflags

При сборке приложения можно передать информацию о версии, дате сборки и коммите с помощью флагов компоновщика:

```bash
go build -ldflags "-X main.buildVersion=v1.0.0 -X main.buildDate=$(date +%Y-%m-%dT%H:%M:%S%z) -X main.buildCommit=$(git rev-parse HEAD)" -o shortener ./cmd/shortener
```

Или с использованием переменных окружения:

```bash
BUILD_VERSION=v1.0.0
BUILD_DATE=$(date +%Y-%m-%dT%H:%M:%S%z)
BUILD_COMMIT=$(git rev-parse HEAD)

go build -ldflags "-X main.buildVersion=${BUILD_VERSION} -X main.buildDate=${BUILD_DATE} -X main.buildCommit=${BUILD_COMMIT}" -o shortener ./cmd/shortener
```

После сборки приложение выведет переданные значения при запуске:

```
Build version: v1.0.0
Build date: 2026-07-21T12:00:00+0300
Build commit: abc123def456...
```

Если значения не переданы, будут использоваться значения по умолчанию `N/A`.