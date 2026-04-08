// Package client provides the [YMClient] aggregator — a single entry point
// to all Yandex Messenger Bot API services.
//
// Create a client with [New] and access services through its fields:
//
//	cs := client.New(ym.Config{Token: os.Getenv("YM_TOKEN")})
//	msg, _ := cs.Messages.SendToChat(ctx, chatID, "hello", nil)
//	chat, _ := cs.Chats.Create(ctx, &chats.ChatCreateRequest{...})
//	updates, _ := cs.Updates.Get(ctx, 100, "")
//
// If you need a pre-configured core [ym.Client] (e.g. with custom HTTP
// transport), use [Wrap] instead:
//
//	cl := ym.NewClientWithHTTP(cfg, customHTTP)
//	cs := client.Wrap(cl)
package client
