// Package users provides operations for retrieving user information from
// Yandex Messenger.
//
// Currently the service supports looking up a user's deep-link by login,
// which can be used to initiate direct conversations.
//
// Create the service via [NewService] with a configured [ym.Client]:
//
//	svc := users.NewService(ymClient)
//	link, err := svc.GetUserLink(ctx, "alice")
package users
