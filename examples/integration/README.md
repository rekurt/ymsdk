# Интеграционные проверки

[English](README.en.md)

Этот пример последовательно вызывает все методы ymsdk через агрегатор `client.New` против реального бота. Настройте переменные окружения и запустите `go run .` или скрипт `run.sh`.

## Обязательные
- `YM_TOKEN` — OAuth-токен бота.

## Опциональные (используются при наличии)
- `YM_CHAT_ID` — чат для отправки сообщений/опросов/файлов.
- `YM_LOGIN` — логин пользователя для личных сообщений и user link.
- `YM_FILE_PATH` — файл для sendFile.
- `YM_IMAGE_PATH` — картинка для sendImage.
- `YM_GALLERY_PATHS` — список картинок через запятую для sendGallery.
- `YM_FILE_ID` — file_id для скачивания через getFile.
- `YM_CREATE_CHAT_NAME` — создать чат/канал; `YM_CREATE_CHAT_CHANNEL=1` для канала.
- `YM_MEMBER_LOGIN` — участник, которого добавить в созданный чат (только для чатов).
- `YM_WEBHOOK_URL` — установить webhook через self.update.

## Что покрывает

- getUserLink — получение ссылок на чат/звонок
- sendText — в чат и в личку
- sendFile / sendImage / sendGallery — файловые операции
- deleteMessage — удаление сообщения
- getFile — скачивание файла
- createPoll / getResults / getVoters — полный цикл опросов
- createChat / updateMembers — создание чата и управление участниками
- self.update — настройка webhook
- getUpdates — получение обновлений

## Запуск
```bash
cd examples/integration
YM_TOKEN=... YM_CHAT_ID=... YM_LOGIN=... YM_FILE_PATH=... go run .
# или
YM_TOKEN=... ./run.sh
```
