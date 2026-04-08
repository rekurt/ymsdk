// Package updates provides mechanisms for receiving real-time updates from
// the Yandex Messenger Bot API.
//
// Two update modes are supported:
//
//   - Long polling via [Service.PollLoop], which continuously fetches new
//     updates and invokes a user-supplied callback.
//   - Webhook handling via the companion webhook example, where the API
//     pushes updates to a user-provided HTTP endpoint.
//
// Create the service via [NewService] with a configured [ym.Client]:
//
//	svc := updates.NewService(ymClient)
//	err := svc.PollLoop(ctx, updates.PollParams{Limit: 100}, func(ctx context.Context, u ym.Update) error {
//	    fmt.Println("got update:", u.UpdateID)
//	    return nil
//	})
package updates
