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

	"github.com/rekurt/ymsdk/config"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/messages"
	"github.com/rekurt/ymsdk/client/ymerrors"
)

func main() {
	token := os.Getenv("YM_TOKEN")
	client := ym.NewClient(ym.Config{
		Token: token,
		ErrorHandling: config.ErrorHandlingConfig{
			RetryStrategy: config.RetryStrategy{MaxAttempts: 3, RetryNetwork: true},
			RateLimitHandling: config.RateLimitHandling{UseRetryAfter: true},
		},
	})

	msgSvc := messages.NewService(client)
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

## Сервисы

- `messages.Service` — текст, файлы, картинки/галереи, delete, getFile.
- `chats.Service` — создание чатов/каналов, обновление участников/подписчиков/админов.
- `users.Service` — получение chat_link/call_link по логину.
- `polls.Service` — создание опросов, результаты, список проголосовавших.
- `updates.Service` — getUpdates и `PollLoop`.
- `self.Service` — `self.update` для webhook_url.
- `middleware` — логирование ошибок через zap.
- Для удобства есть агрегатор `sdk.ClientSet` с уже сконструированными сервисами (`sdk.New(cfg)`).

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
import "github.com/rekurt/ymsdk/client"

cs := sdk.New(ym.Config{Token: "..."})
msg, _ := cs.Messages.SendToChat(ctx, "chat-id", "hi", nil)
_ = cs.Polls.Create(ctx, &polls.CreatePollRequest{ChatID: ptr("chat-id"), Title: "Q?", Answers: []string{"A","B"}})
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

## Тесты

```bash
go test ./...
```
