package stanclient

import stan "github.com/nats-io/stan.go"

// Subscriber implements everything Subscribe* needs
type Subscriber interface {
	MsgHandler() stan.MsgHandler
	Subject() string
	DurableName() string
	Name() string
}
