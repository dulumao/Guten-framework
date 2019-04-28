package observer

import (
	"github.com/dulumao/Guten-utils/os/event"
)

var Dispatcher event.Dispatcher

func New() {
	Dispatcher = event.New()
}
