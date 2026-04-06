# Yandex Messenger Go SDK (ymsdk)

[English](README.en.md)

Легковесный Go-клиент для Yandex Messenger Bot API с типобезопасными моделями, встроенным retry и сервисами для всех основных методов API. Документация: https://pkg.go.dev/github.com/rekurt/ymsdk

## Установка

```bash
go get github.com/rekurt/ymsdk
```

## Быстрый старт

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/messages"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

func main() {
	token := os.Getenv("YM_TOKEN")
	cl := ym.NewClient(ym.Config{
		Token: token,
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy:     ymerrors.RetryStrategy{MaxAttempts: 3, RetryNetwork: true},
			RateLimitHandling: ymerrors.RateLimitHandling{UseRetryAfter: true},
		},
	})

	msgSvc := messages.NewService(cl)
	msg, err := msgSvc.SendToChat(context.Background(), "chat-id", "hello", nil)
	if err != nil {
		handleErr(err)
		return
	}
	fmt.Println("sent message:", msg.ID)
}

func handleErr(err error) {
	var apiErr *ymerrors.APIError
	if errors.As(err, &apiErr) {
		fmt.Printf("API error kind=%d http=%d desc=%s\n", apiErr.Kind, apiErr.HTTPStatus, apiErr.Description)
		if errors.Is(err, ymerrors.ErrRateLimited) && apiErr.RetryAfter > 0 {
			fmt.Printf("retry after: %s\n", apiErr.RetryAfter)
		}
		return
	}
	fmt.Println("unexpected error:", err)
}
```

См. примеры в `examples/basic_send`, `examples/poller`, `examples/poll_bot`, `examples/integration`.

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
    ├── messages/       # Текст, файлы, картинки, галереи, удаление
    ├── chats/          # Создание чатов, управление участниками
    ├── users/          # Ссылки на чат/звонок пользователя
    ├── polls/          # Опросы, результаты, голоса
    ├── updates/        # getUpdates и PollLoop
    ├── self/           # Управление webhook_url бота
    └── files/          # Низкоуровневая отправка файлов
middleware/             # Логирование через zap
```

## Сервисы

- `messages.Service` — текст, файлы, картинки/галереи, delete, getFile.
- `chats.Service` — создание чатов/каналов, обновление участников/подписчиков/админов.
- `users.Service` — получение chat_link/call_link по логину.
- `polls.Service` — создание опросов, результаты, список проголосовавших.
- `updates.Service` — getUpdates и `PollLoop`.
- `self.Service` — `self.update` для webhook_url.
- `middleware` — логирование ошибок через zap.
- Для удобства есть агрегатор `client.YMClient` с уже сконструированными сервисами (`client.New(cfg)`).

## Обработка ошибок

- Все API-ошибки — `*ymerrors.APIError`; используйте `errors.As`.
- Rate limit: `errors.Is(err, ymerrors.ErrRateLimited)` + `RetryAfter`.
- Авторизация: `ErrInvalidToken`/`ErrUnauthorized`.
- Сетевые: `KindNetwork` или `net.Error`, если включён `RetryNetwork`.

## Конфигурация

`ym.Config`:

- `BaseURL` — endpoint (по умолчанию production).
- `Token` — OAuth-токен.
- `ErrorHandling`:
  - `RetryStrategy`: `MaxAttempts`, `InitialBackoff`, `MaxBackoff`, `RetryHTTP`, `RetryNetwork`.
  - `RateLimitHandling`: `UseRetryAfter`, `DefaultBackoff`.
- `UpdatesMode`: `polling`/`webhook` (для явной фиксации режима).

## Запуск примеров

- `examples/basic_send` — отправка текста в чат/логин, обработка ошибок.
- `examples/poller` — опрос обновлений с respect к rate limit.
- `examples/poll_bot` — создание опроса и чтение обновлений.
- `examples/integration` — скрипт, проходящий по всем методам SDK (настройка через env).
- `examples/webhook` — минимальный HTTP-приемник webhook (для режима webhook).

### Быстро через агрегатор

```go
import (
	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/polls"
)

cs := client.New(ym.Config{Token: "..."})
msg, _ := cs.Messages.SendToChat(ctx, "chat-id", "hi", nil)
_ = cs.Polls.Create(ctx, &polls.CreatePollRequest{
	ChatID:  ym.Ptr(ym.ChatID("chat-id")),
	Title:   "Q?",
	Answers: []string{"A", "B"},
})
```

Запуск интеграции:
```bash
cd examples/integration
YM_TOKEN=... YM_CHAT_ID=... YM_LOGIN=... YM_FILE_PATH=... go run .
# или: YM_TOKEN=... ./run.sh
```

Запуск webhook-примера:
```bash
cd examples/webhook
YM_TOKEN=... YM_PORT=8080 go run .
```

## Версионирование

Проект следует [Semantic Versioning](https://semver.org/). Для установки конкретной версии:

```bash
go get github.com/rekurt/ymsdk@v0.1.0
```

## Тесты

```bash
go test ./...
```
