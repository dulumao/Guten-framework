package events

import (
	"github.com/dulumao/Guten-utils/os/event"
)

type Events struct {
	Name     string
	Priority int
	Callback func(event *event.Event) error
}
