# Integration Checks

[Русская версия](README.md)

This example exercises all ymsdk methods against a real bot. Configure environment variables, then run `go run .` or `run.sh`.

## Required
- `YM_TOKEN` — bot OAuth token.

## Optional (used when present)
- `YM_CHAT_ID` — chat to send messages/polls/files.
- `YM_LOGIN` — user login for direct messages/user link.
- `YM_FILE_PATH` — file to send via sendFile.
- `YM_IMAGE_PATH` — single image for sendImage.
- `YM_GALLERY_PATHS` — comma-separated list of images for sendGallery.
- `YM_FILE_ID` — file_id to fetch via getFile.
- `YM_CREATE_CHAT_NAME` — create chat/channel; set `YM_CREATE_CHAT_CHANNEL=1` for channel.
- `YM_MEMBER_LOGIN` — member to add when creating chat (for chats).
- `YM_WEBHOOK_URL` — set webhook via self.update.

## Run
```bash
cd examples/integration
YM_TOKEN=... YM_CHAT_ID=... YM_LOGIN=... YM_FILE_PATH=... go run .
# or
YM_TOKEN=... ./run.sh
```

The script logs each step (text, files/images/gallery, delete, getFile, polls create/results/voters, chat create/members, getUserLink, self.update, getUpdates) so you can verify end-to-end behavior quickly.
