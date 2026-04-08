// Package polls provides operations for creating and managing polls in
// Yandex Messenger chats.
//
// The service supports poll creation, retrieving poll results, and
// paginating through individual voter responses.
//
// Create the service via [NewService] with a configured [ym.Client]:
//
//	svc := polls.NewService(ymClient)
//	msg, err := svc.Create(ctx, &polls.CreatePollRequest{
//	    ChatID:   chatID,
//	    Question: "Lunch?",
//	    Answers:  []string{"Pizza", "Sushi"},
//	})
//
// Use [Service.GetResults] to fetch aggregated poll results and
// [Service.GetAllVoters] to collect all individual votes with automatic
// pagination.
package polls
