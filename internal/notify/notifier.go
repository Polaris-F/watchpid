package notify

import "context"

type Message struct {
	Title string
	Body  string
}

type Notifier interface {
	Name() string
	Send(ctx context.Context, msg Message) error
}
