package conversation

// Config is struct with configuration parameters for a conversation
type Config struct {
	MaxMessageQueue int // the maximum size of unporcessed message queue for a conversation
	TimeoutMinutes  int // timeout for a user's input in minutes
}
