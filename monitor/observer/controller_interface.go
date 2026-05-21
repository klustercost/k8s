package observer

import "context"

type Controller interface {
	Run(context.Context, int) error
	FriendlyName() string
}
