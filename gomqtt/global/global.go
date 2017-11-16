package global

// topic types
const (
	// Master Topic
	MTopic = 1000
	// Private Chat Topic
	PChatTopic = 2000

	// Group Chat Topic
	GChatTopic = 4000

	// Singe Push Topic
	SPushTopic = 3000

	// Broadcast Push Topic
	BPushTopic = 5000
)

// payload proto types
const (
	PayloadText     = 1
	PayloadJson     = 2
	PayloadProtobuf = 3
)

// message types
const (
	// PChatPush
	PrivateChat = 11000
)
