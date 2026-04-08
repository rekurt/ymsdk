# Yandex Messenger Go SDK (ymsdk)

[English](README.en.md)

[![CI](https://github.com/rekurt/ymsdk/actions/workflows/ci.yml/badge.svg)](https://github.com/rekurt/ymsdk/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/rekurt/ymsdk.svg)](https://pkg.go.dev/github.com/rekurt/ymsdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/rekurt/ymsdk)](https://goreportcard.com/report/github.com/rekurt/ymsdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rekurt/ymsdk)](go.mod)
[![codecov](https://codecov.io/gh/rekurt/ymsdk/branch/master/graph/badge.svg)](https://codecov.io/gh/rekurt/ymsdk)

Легковесный Go-клиент для Yandex Messenger Bot API с типобезопасными моделями, встроенным retry и сервисами для всех основных методов API. Документация: https://pkg.go.dev/github.com/rekurt/ymsdk

## Возможности

- **Типобезопасные модели** — `ChatID`, `UserLogin`, `MessageID` и другие типы предотвращают ошибки на этапе компиляции
- **Автоматический retry** — экспоненциальный backoff с настраиваемой стратегией повторных попыток
- **Rate limit** — автоматическое соблюдение `Retry-After` заголовков API
- **Сервис-ориентированная архитектура** — отдельные пакеты для сообщений, чатов, опросов, обновлений, файлов и пользователей
- **Polling и Webhooks** — два режима получения обновлений
- **Debug-логирование** — структурированные логи через `zap` с HTTP-инспекцией
- **Минимум зависимостей** — только `go.uber.org/zap`
- **Полное покрытие API** — все основные методы Yandex Messenger Bot API

## Установка

```bash
go get github.com/rekurt/ymsdk
```

## Быстрый старт

### Через агрегатор (рекомендуется)

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

func main() {
	cs := client.New(ym.Config{
		Token: os.Getenv("YM_TOKEN"),
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy:     ymerrors.RetryStrategy{MaxAttempts: 3, RetryNetwork: true},
			RateLimitHandling: ymerrors.RateLimitHandling{UseRetryAfter: true},
		},
	})

	msg, err := cs.Messages.SendToChat(context.Background(), "chat-id", "hello", nil)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("sent message:", msg.ID)
}
```

### Через отдельные сервисы

```go
cl := ym.NewClient(ym.Config{Token: os.Getenv("YM_TOKEN")})
msgSvc := messages.NewService(cl)
pollSvc := polls.NewService(cl)

msg, _ := msgSvc.SendToChat(ctx, "chat-id", "hello", nil)
```

## Архитектура

```
client/
├── sdk.go              # YMClient — агрегатор со всеми сервисами
└── ym/                 # Ядро SDK
    ├── client.go       # HTTP-клиент с retry/rate-limit логикой
    ├── types.go        # Общие типы (Chat, Message, Update, …)
    ├── ptr.go          # Хелпер ym.Ptr[T] для optional-полей
    ├── validate.go     # Общая валидация получателя
    ├── ymerrors/       # Типы ошибок и конфигурация
    ├── messages/       # Текст, файлы, картинки, галереи, удаление, getFile
    ├── chats/          # Создание чатов/каналов, управление участниками
    ├── users/          # Ссылки на чат/звонок пользователя
    ├── polls/          # Опросы: создание, результаты, голоса
    ├── updates/        # getUpdates, GetUpdates и PollLoop
    ├── self/           # Управление webhook_url бота
    └── files/          # Низкоуровневая отправка файлов (byte[])
middleware/             # Логирование через zap
├── logging.go          # LogError, LogUpdateWithRawData, WithRequestID
├── debug.go            # DebugLogger с уровнями (Silent → Debug)
└── http_logger.go      # HTTP-обёртка для логирования request/response
```

## Сервисы

| Сервис | Описание |
|--------|----------|
| `cs.Messages` | Текстовые сообщения, файлы, картинки, галереи, удаление, скачивание файлов |
| `cs.Chats` | Создание чатов/каналов, добавление/удаление участников, подписчиков, админов |
| `cs.Users` | Получение chat_link / call_link по логину |
| `cs.Polls` | Создание опросов, результаты, постраничный список голосов, GetAllVoters |
| `cs.Updates` | getUpdates (raw + typed), PollLoop для непрерывного опроса |
| `cs.Self` | self.update для настройки webhook_url |
| `cs.Files` | Низкоуровневая отправка файлов через byte[] |

Для удобства есть агрегатор `client.YMClient` с уже сконструированными сервисами:
- `client.New(cfg)` — создание с новым HTTP-клиентом
- `client.Wrap(cl)` — обёртка над существующим `ym.Client`

## Обработка ошибок

```go
var apiErr *ymerrors.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("kind=%d http=%d desc=%s request_id=%s\n",
        apiErr.Kind, apiErr.HTTPStatus, apiErr.Description, apiErr.RequestID)

    if errors.Is(err, ymerrors.ErrRateLimited) && apiErr.RetryAfter > 0 {
        time.Sleep(apiErr.RetryAfter)
    }
}
```

- Все API-ошибки — `*ymerrors.APIError`; используйте `errors.As`.
- Rate limit: `errors.Is(err, ymerrors.ErrRateLimited)` + `RetryAfter`.
- Авторизация: `ErrInvalidToken` (403) / `ErrUnauthorized` (401).
- Сетевые: `KindNetwork` (5xx) или `net.Error`, если включён `RetryNetwork`.

## Конфигурация

```go
cfg := ym.Config{
    BaseURL: "",  // по умолчанию production endpoint
    Token:   os.Getenv("YM_TOKEN"),
    ErrorHandling: ymerrors.ErrorHandlingConfig{
        RetryStrategy: ymerrors.RetryStrategy{
            MaxAttempts:    3,               // до 3 попыток
            InitialBackoff: 500 * time.Millisecond,
            MaxBackoff:     10 * time.Second,
            RetryNetwork:   true,            // повторять при сетевых ошибках
            RetryHTTP:      []int{500, 502, 503, 504},
        },
        RateLimitHandling: ymerrors.RateLimitHandling{
            UseRetryAfter:  true,            // уважать Retry-After заголовок
            DefaultBackoff: time.Second,
        },
    },
    UpdatesMode: ymerrors.UpdatesModePolling, // "polling" или "webhook"
}
```

## Debug-логирование

Для отладки HTTP-запросов и ответов используйте middleware:

```go
import (
    "github.com/rekurt/ymsdk/client"
    "github.com/rekurt/ymsdk/client/ym"
    "github.com/rekurt/ymsdk/middleware"
)

logger, _ := zap.NewDevelopmentConfig().Build()
debugLogger := middleware.NewDebugLogger(logger, middleware.LogLevelDebug)
loggedHTTP := middleware.NewHTTPLogger(&http.Client{Timeout: 15 * time.Second}, debugLogger)

ymClient := ym.NewClientWithHTTP(cfg, loggedHTTP)
cs := client.Wrap(ymClient)
```

Подробнее — `middleware/README.md` и `examples/debug_logger`.

## Примеры

| Пример | Описание |
|--------|----------|
| `examples/basic_send` | Отправка текста в чат/логин, reply-to, mark-important, обработка ошибок |
| `examples/poller` | Непрерывный опрос обновлений через PollLoop, обработка типов (текст, файлы, стикеры, пересланные) |
| `examples/poll_bot` | Создание опроса, GetResults, GetAllVoters, чтение обновлений |
| `examples/webhook` | HTTP-приёмник webhook с валидацией секрета, graceful shutdown, echo-бот |
| `examples/debug_logger` | HTTP-логирование запросов/ответов, обработка обновлений без сообщений |
| `examples/integration` | Полный обход всех методов SDK (настройка через env) |

### Запуск примеров

```bash
# Отправка сообщения
cd examples/basic_send
YM_TOKEN=... go run . -chat "chat-id" -text "hello"

# Polling обновлений
cd examples/poller
YM_TOKEN=... go run .

# Опрос-бот
cd examples/poll_bot
YM_TOKEN=... YM_CHAT_ID=... go run .

# Webhook-сервер
cd examples/webhook
YM_TOKEN=... YM_WEBHOOK_SECRET=... YM_PORT=8080 go run .

# Debug-логирование
cd examples/debug_logger
YM_TOKEN=... go run .

# Полная интеграция
cd examples/integration
YM_TOKEN=... YM_CHAT_ID=... YM_LOGIN=... go run .
```

## Версионирование

Проект следует [Semantic Versioning](https://semver.org/). Для установки конкретной версии:

```bash
go get github.com/rekurt/ymsdk@v0.1.0
```

## Тесты

```bash
# Все тесты
go test ./...

# Линтинг (50+ линтеров)
golangci-lint run --config .golangci.yml
```
