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

}

func (c *Crawler) CloseSpider() {

}

func (c *Crawler) EnqueueScrape() {

}

func (c *Crawler) Scrape(response *Response, request *Request) {
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
				"action": "dropped",
				"item": item,
			}).Warnf("Dropped item %s", item)
			ItemDropped.Pub(c.Spider, response, item)
		} else if err != nil {
			c.Logger.WithError(err).Errorf("Processing item %s", item)
		} else {
			c.Logger.WithFields(logrus.Fields{
				"action": "scraped",
				"item": resultItem,
				"src": response,
			}).Debugf("Scraped item %s from %s", resultItem, response)
		}
	}, nil)
}
