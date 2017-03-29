package spy

import (
	"reflect"
	"github.com/Sirupsen/logrus"
	"github.com/Jeffail/tunny"
)

type Crawler struct {
	*Config
	*logrus.Logger
	*Stats

	Concurrency int

	Spider ISpider
	Scheduler IScheduler
	Fetcher IFetcher
	*SpiderMiddlewareManager
	*ItemPipelineManager

	crawling bool
	Paused bool

	*tunny.WorkPool
}

func NewCrawler(spider ISpider, scheduler IScheduler) *Crawler {
	concurrency := 100

	return &Crawler{
		Spider: spider,
		Scheduler: scheduler,
		Concurrency: concurrency,
		WorkPool: tunny.CreatePoolGeneric(concurrency)
	}
}

func (c *Crawler) Craw() {
	// TODO: signal handling for SIGTERM, SIGINT, SIGBREAK

	if c.crawling {
		panic("Crawling already taking place")
	}
	c.crawling = true

	c.WorkPool.Open()
}

func (c *Crawler) Stop() {
	c.WorkPool.Close()
}

func (c *Crawler) OpenSpider() {
	startRequests, err := c.SpiderMiddlewareManager.ProcessStartRequests(c.Spider.StartRequests(), c.Spider)

	c.Scheduler.Open(c.Spider)

	SpiderOpened.Pub(c.Spider)

	c.nextRequest()
}

func (c *Crawler) CloseSpider() {

}

func (c *Crawler) nextRequest() {
	if c.Paused {
		return
	}

	for !c.needsBackout() {
		request := c.Scheduler.NextRequest()
		if request == nil {
			break
		}
		c.Fetch(request)
	}

}

func (c *Crawler) needsBackout() {
	return !c.crawling // TODO
}

func (c *Crawler) Fetch(request *Request) {
	result, err := c.Fetcher.Fetch(request, c.Spider)
	if err != nil {
		c.EnqueueScrape(nil, err, request)
		return
	}
	if result.Response != nil {
		result.Response.Request = request // tie request to response received
		c.Logger.WithFields(logrus.Fields{
			"action": "crawl",
			"status": result.Response.StatusCode,
			"request": request,
		}).Debugf("Crawled request %s, status %d", request, result.Response.StatusCode)
		ResponseReceived.Pub(c.Spider, request, result.Response)

		c.EnqueueScrape(result.Response, nil, request)
	} else { // fetcher can return request, i.e., redirect
		c.Fetch(result.Request)
	}
}

func (c *Crawler) EnqueueScrape(response *Response, err error, request *Request) {

}

func (c *Crawler) Scrape(response *Response, err error, request *Request) {
	result, err := c.SpiderMiddlewareManager.ScrapeResponse(request, response, c.Spider)
	if err == nil {
		for _, req := range result.Requests {
			c.processSpiderRequest(req)
		}
		for _, item := range result.Items {
			c.processSpiderItem(item)
		}

	} else {
		if err == ErrSpiderClosed {
			return
		}
		c.Logger.WithError(err).Errorf("Processing request %s (referer: %s)", request, request.Header.Get("Referer"))
		SpiderError.Pub(c.Spider, response, err)
		c.Stats.Inc("SpiderError/"+reflect.TypeOf(err).Name())
	}
}

func (c *Crawler) processSpiderRequest(request *Request) {
	c.WorkPool.SendWorkAsync(func() {

	}, nil)
}

func (c *Crawler) processSpiderItem(item *Item, response *Response) {
	c.WorkPool.SendWorkAsync(func() {
		resultItem, err := c.ItemPipelineManager.ProcessItem(item, c.Spider)

		if err == ErrItemDropped {
			c.Logger.WithFields(logrus.Fields{
				"action": "drop",
				"item": item,
			}).Warnf("Dropped item %s", item)
			ItemDropped.Pub(c.Spider, response, item)
		} else if err != nil {
			c.Logger.WithError(err).Errorf("Processing item %s", item)
		} else {
			c.Logger.WithFields(logrus.Fields{
				"action": "scrap",
				"item": resultItem,
				"src": response,
			}).Debugf("Scraped item %s from %s", resultItem, response)
		}
	}, nil)
}
