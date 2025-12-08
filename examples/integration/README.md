# Интеграционные проверки

[English](README.en.md)

Этот пример последовательно вызывает все методы ymsdk против реального бота. Настройте переменные окружения и запустите `go run .` или скрипт `run.sh`.

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

## Запуск
```bash
cd examples/integration
YM_TOKEN=... YM_CHAT_ID=... YM_LOGIN=... YM_FILE_PATH=... go run .
# или
YM_TOKEN=... ./run.sh
```

Скрипт логирует шаги (текст, файлы/картинки/галерея, delete, getFile, опросы create/results/voters, создание чата и участники, getUserLink, self.update, getUpdates), чтобы быстро проверить работу всего SDK.
