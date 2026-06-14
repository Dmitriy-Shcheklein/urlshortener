# go-musthave-shortener-tpl

Шаблон репозитория для трека «Сервис сокращения URL».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-shortener-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Результаты профилирования

### Бенчмарки

| Метрика | До оптимизации | После оптимизации | Изменение |
|---------|----------------|-------------------|-----------|
| `ns/op` | 43,053 | 33,100 | **-23%** |
| `B/op` | 42,842 | 26,500 | **-38%** |
| `allocs/op` | 410 | 203 | **-50%** |

### Сравнение heap-профилей (pprof -diff_base)

```
File: shortener
Type: inuse_space
Time: 2026-06-14 16:21:19 +05
Showing nodes accounting for 1544.54kB, 29.98% of 5151.30kB total
Dropped 2 nodes (cum <= 25.76kB)
      flat  flat%   sum%        cum   cum%
    1028kB 19.96% 19.96%     1028kB 19.96%  bufio.NewWriterSize (inline)
  516.64kB 10.03% 29.99%  1028.65kB 19.97%  github.com/go-playground/validator/v10.New
 -512.14kB  9.94% 20.04%  -512.14kB  9.94%  github.com/go-playground/validator/v10.New.func1
  512.05kB  9.94% 29.98%   512.05kB  9.94%  github.com/go-playground/validator/v10.(*Validate).parseFieldTagsRecursive
 -512.02kB  9.94% 20.04%  -512.02kB  9.94%  syscall.anyToSockaddr
  512.01kB  9.94% 29.98%   512.01kB  9.94%  github.com/go-playground/validator/v10.wrapFunc (inline)
         0     0% 29.98%  1028.55kB 19.97%  github.com/Dmitriy-Shcheklein/urlshortener/internal/handler/shortener.(*Handler).CreateFromJSONBody
         0     0% 29.98%  1028.55kB 19.97%  github.com/Dmitriy-Shcheklein/urlshortener/internal/middlewares.(*AppMiddleware).Auth-fm.(*AppMiddleware).Auth.func1
         0     0% 29.98%  1028.55kB 19.97%  github.com/Dmitriy-Shcheklein/urlshortener/internal/middlewares.(*AppMiddleware).WithGzip-fm.(*AppMiddleware).WithGzip.func1
         0     0% 29.98%  1028.55kB 19.97%  github.com/Dmitriy-Shcheklein/urlshortener/internal/middlewares.(*AppMiddleware).WithLogging-fm.(*AppMiddleware).WithLogging.func1
         0     0% 29.98%  1028.55kB 19.97%  github.com/go-chi/chi.(*Mux).ServeHTTP
         0     0% 29.98%  1028.55kB 19.97%  github.com/go-chi/chi.(*Mux).routeHTTP
         0     0% 29.98%  1028.55kB 19.97%  github.com/go-chi/chi/middleware.RealIP.func1
         0     0% 29.98%  1028.55kB 19.97%  github.com/go-chi/chi/middleware.Recoverer.func1
         0     0% 29.98%  1028.55kB 19.97%  github.com/go-chi/chi/middleware.RequestID.func1
         0     0% 29.98%   512.05kB  9.94%  github.com/go-playground/validator/v10.(*Validate).extractStructCache
         0     0% 29.98%   512.05kB  9.94%  github.com/go-playground/validator/v10.(*validate).validateStruct
         0     0% 29.98%  -512.02kB  9.94%  internal/poll.(*FD).Accept
         0     0% 29.98%  -512.02kB  9.94%  internal/poll.accept
         0     0% 29.98%  1028.55kB 19.97%  main.main.Timeout.func3.1
         0     0% 29.98%  -512.02kB  9.94%  main.main.func2
         0     0% 29.98%  -512.02kB  9.94%  net.(*TCPListener).Accept
         0     0% 29.98%  -512.02kB  9.94%  net.(*TCPListener).accept
         0     0% 29.98%  -512.02kB  9.94%  net.(*netFD).accept
         0     0% 29.98%  -512.02kB  9.94%  net/http.(*Server).ListenAndServe
         0     0% 29.98%  -512.02kB  9.94%  net/http.(*Server).Serve
         0     0% 29.98%  2056.56kB 39.92%  net/http.(*conn).serve
         0     0% 29.98%  1028.55kB 19.97%  net/http.HandlerFunc.ServeHTTP
         0     0% 29.98%     1028kB 19.96%  net/http.newBufioWriterSize
         0     0% 29.98%  1028.55kB 19.97%  net/http.serverHandler.ServeHTTP
         0     0% 29.98%  -512.14kB  9.94%  sync.(*Pool).Get
         0     0% 29.98%  -512.02kB  9.94%  syscall.Accept
```

### Оптимизации

1. **validator.New() singleton** — экземпляр валидатора создаётся один раз в конструкторе `Handler`, а не на каждый запрос
2. **json.NewDecoder → json.Unmarshal** — убрано создание JSON-декодера на каждый запрос

### Обнаруженные проблемы

| Проблема | Память | Статус                                                            |
|----------|--------|-------------------------------------------------------------------|
| `validator.New()` на запрос | 342 MB | ✅ Исправлено                                                      |
| `json.NewDecoder()` на запрос | 24 MB | ✅ Исправлено                                                      |
| Auditor goroutines | 23 MB | Не оптимально, решено оставить, так как неясна критичность аудита |
