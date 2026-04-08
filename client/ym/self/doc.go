// Package self provides operations for managing the bot's own settings in
// Yandex Messenger, such as configuring webhook URLs.
//
// Create the service via [NewService] with a configured [ym.Client]:
//
//	svc := self.NewService(ymClient)
//	bot, err := svc.Update(ctx, &self.SelfUpdateRequest{
//	    WebhookURL: "https://example.com/webhook",
//	})
package self
