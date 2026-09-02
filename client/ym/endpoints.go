package ym

// API endpoint paths for the Yandex Messenger Bot API.
//
// Paths are reproduced exactly as documented, including the trailing slash.
// Two endpoints are documented without it — [EndpointSelfGet] and
// [EndpointUsersGetUserLink] — so the constants differ from the rest on
// purpose. Always reference these constants instead of writing a literal:
// hand-written paths previously drifted apart, with sendFile being sent both
// with and without the trailing slash from two different packages.
//
// See https://yandex.ru/dev/messenger/doc/ru/ for the method reference.
const (
	// Chats.

	// EndpointChatsCreate creates a chat or a channel.
	EndpointChatsCreate = "/bot/v1/chats/create/"
	// EndpointChatsGet lists chats and channels the bot belongs to.
	EndpointChatsGet = "/bot/v1/chats/get/"
	// EndpointChatsGetChat returns information about a single chat or channel.
	EndpointChatsGetChat = "/bot/v1/chats/getChat/"
	// EndpointChatsGetMembers lists members of a chat or channel.
	EndpointChatsGetMembers = "/bot/v1/chats/getMembers/"
	// EndpointChatsUpdateMembers adds, promotes or removes chat members.
	EndpointChatsUpdateMembers = "/bot/v1/chats/updateMembers/"

	// Messages.

	// EndpointMessagesCreatePoll creates a poll.
	EndpointMessagesCreatePoll = "/bot/v1/messages/createPoll/"
	// EndpointMessagesDelete deletes a message.
	EndpointMessagesDelete = "/bot/v1/messages/delete/"
	// EndpointMessagesGetFile downloads a previously uploaded file.
	EndpointMessagesGetFile = "/bot/v1/messages/getFile/"
	// EndpointMessagesGetReactions returns reactions on a message.
	EndpointMessagesGetReactions = "/bot/v1/messages/getReactions/"
	// EndpointMessagesGetUpdates fetches pending updates (polling).
	EndpointMessagesGetUpdates = "/bot/v1/messages/getUpdates/"
	// EndpointMessagesPin pins a message in a chat.
	EndpointMessagesPin = "/bot/v1/messages/pin/"
	// EndpointMessagesSendFile uploads and sends a file.
	EndpointMessagesSendFile = "/bot/v1/messages/sendFile/"
	// EndpointMessagesSendGallery uploads and sends an album of images.
	EndpointMessagesSendGallery = "/bot/v1/messages/sendGallery/"
	// EndpointMessagesSendImage uploads and sends an image.
	EndpointMessagesSendImage = "/bot/v1/messages/sendImage/"
	// EndpointMessagesSendReaction sets or removes the bot's reaction.
	EndpointMessagesSendReaction = "/bot/v1/messages/sendReaction/"
	// EndpointMessagesSendSticker sends a sticker.
	EndpointMessagesSendSticker = "/bot/v1/messages/sendSticker/"
	// EndpointMessagesSendSystemMessage sends a system message.
	EndpointMessagesSendSystemMessage = "/bot/v1/messages/sendSystemMessage/"
	// EndpointMessagesSendText sends a text message, or edits one when message_id is set.
	EndpointMessagesSendText = "/bot/v1/messages/sendText/"
	// EndpointMessagesSendTyping sends a typing or processing indicator.
	EndpointMessagesSendTyping = "/bot/v1/messages/sendTyping/"
	// EndpointMessagesShareFile sends an already uploaded file by its file_id.
	EndpointMessagesShareFile = "/bot/v1/messages/shareFile/"
	// EndpointMessagesShareGallery sends an album of already uploaded images.
	EndpointMessagesShareGallery = "/bot/v1/messages/shareGallery/"
	// EndpointMessagesShareImage sends an already uploaded image by its file_id.
	EndpointMessagesShareImage = "/bot/v1/messages/shareImage/"
	// EndpointMessagesUnpin unpins a message.
	EndpointMessagesUnpin = "/bot/v1/messages/unpin/"

	// Polls.

	// EndpointPollsGetResults returns aggregated poll results.
	EndpointPollsGetResults = "/bot/v1/polls/getResults/"
	// EndpointPollsGetVoters returns a page of poll voters.
	EndpointPollsGetVoters = "/bot/v1/polls/getVoters/"

	// Self.

	// EndpointSelfGet returns information about the bot. Documented without a trailing slash.
	EndpointSelfGet = "/bot/v1/self/get"
	// EndpointSelfUpdate updates the bot's settings.
	EndpointSelfUpdate = "/bot/v1/self/update/"

	// Users.

	// EndpointUsersGetUserLink returns chat and call deep links for a user.
	// Documented without a trailing slash.
	EndpointUsersGetUserLink = "/bot/v1/users/getUserLink"
)
