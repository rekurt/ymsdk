package ym

import (
	"crypto/rand"
	"encoding/hex"
)

// NewPayloadID returns a random idempotency key suitable for the payload_id
// request field.
//
// The API treats two requests carrying the same payload_id as duplicates and
// delivers only one. Since a retried request replays an identical body, the
// same key travels with every attempt, which is what makes retrying a POST safe:
// a sendText that times out mid-flight and is retried produces one message
// rather than two.
//
// Services fill this in automatically unless [Config.DisableAutoPayloadID] is
// set or the caller supplied a key of their own.
func NewPayloadID() string {
	var buf [16]byte
	// crypto/rand.Read never reports an error — it panics if the system entropy
	// source fails — so the buffer is always populated.
	_, _ = rand.Read(buf[:])

	return hex.EncodeToString(buf[:])
}
