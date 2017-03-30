package spy

import (
	"github.com/asaskevich/EventBus"
)

var bus = EventBus.New()

type Event string

const (
	CrawlerStarted     Event = "CrawlerStarted"
	CrawlerStopped     Event = "CrawlerStopped"
	SpiderOpened       Event = "SpiderOpened "
	SpiderIdle         Event = "SpiderIdle"
	SpiderClosed       Event = "SpiderClosed"
	SpiderError        Event = "SpiderError"
	RequestScheduled   Event = "RequestScheduled"
	RequestDropped     Event = "RequestDropped"
	ResponseReceived   Event = "ResponseReceived"
	ResponseDownloaded Event = "ResponseDownloaded"
	ItemScraped        Event = "ItemScraped"
	ItemDropped        Event = "ItemDropped"
)

func (e Event) SubAsync(fn interface{}) {
	panic(bus.SubscribeAsync(string(e), fn, false))
}

func (e Event) Sub(fn interface{}) {
	panic(bus.Subscribe(string(e), fn))
}

func (e Event) Unsub(fn interface{}) {
	panic(bus.Unsubscribe(string(e), fn))
}

func (e Event) Pub(args ...interface{}) {
	bus.Publish(string(e), args...)
}
