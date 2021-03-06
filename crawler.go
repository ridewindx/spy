package spy

import (
	"github.com/Jeffail/tunny"
	"github.com/Sirupsen/logrus"
	"github.com/ridewindx/crumb/concurrency"
	"reflect"
)

type Crawler struct {
	*Config
	*logrus.Logger
	*Stats

	Concurrency int

	Spider    ISpider
	Scheduler IScheduler
	Fetcher   IFetcher
	*SpiderMiddlewareManager
	*ItemPipelineManager

	crawling bool

	*tunny.WorkPool
	*concurrency.Worker
}

func NewCrawler(spider ISpider, scheduler IScheduler) *Crawler {
	concurrencyLimit := 100

	return &Crawler{
		Spider:      spider,
		Scheduler:   scheduler,
		Concurrency: concurrencyLimit,
		WorkPool:    tunny.CreatePoolGeneric(concurrencyLimit),
		Worker:      concurrency.NewWorker(),
	}
}

func (c *Crawler) Start() {
	// TODO: signal handling for SIGTERM, SIGINT, SIGBREAK

	if c.crawling {
		panic("Crawling already taking place")
	}
	c.crawling = true

	c.WorkPool.Open()

	c.openSpider()

	CrawlerStarted.Pub(c)
}

func (c *Crawler) Stop() {
	c.crawling = false
	c.closeSpider()
	c.WorkPool.Close()
	c.Worker.Stop()
	CrawlerStopped.Pub(c)
}

func (c *Crawler) openSpider() {
	c.Logger.WithField("spider", c.Spider.String()).Info("Opening spider")

	startRequests, err := c.SpiderMiddlewareManager.ProcessStartRequests(c.Spider.StartRequests(), c.Spider)
	if err != nil {
		c.Logger.WithError(err).WithField("spider", c.Spider.String()).Panicf("Processing starting requests")
	}

	c.Stats.Open(c.Spider)
	c.Scheduler.Open(c.Spider)
	c.ItemPipelineManager.Open(c.Spider)

	SpiderOpened.Pub(c.Spider)

	c.scheduleRequests(startRequests)

	c.Logger.WithField("spider", c.Spider.String()).Info("Opened spider")
}

func (c *Crawler) closeSpider() {
	c.Logger.WithField("spider", c.Spider.String()).Info("Closing spider")

	c.ItemPipelineManager.Close(c.Spider)
	c.Scheduler.Close(c.Spider)
	c.Stats.Close(c.Spider)

	SpiderClosed.Pub(c.Spider)

	c.Logger.WithField("spider", c.Spider.String()).Info("Closed spider")
}

func (c *Crawler) Pause() {
	c.Worker.Pause()
}

func (c *Crawler) Resume() {
	c.Worker.Resume()
}

func (c *Crawler) enqueueRequest(request *Request) {
	RequestScheduled.Pub(c.Spider, request)
	ok := c.Scheduler.EnqueueRequest(request)
	if !ok {
		RequestDropped.Pub(c.Spider, request)
	}
}

func (c *Crawler) scheduleRequests(startRequests []*Request) {
	c.Worker.Start(func(sentry *concurrency.Sentry) {
		// breadth first
		for _, request := range startRequests {
			c.enqueueRequest(request)
		}

		for sentry.Stopped() && !c.needsBackout() {
			sentry.Sleep()

			request := c.Scheduler.NextRequest()
			if request == nil {
				break
			}
			c.fetch(request)
		}
	})
}

func (c *Crawler) needsBackout() bool {
	return !c.crawling // TODO
}

func (c *Crawler) fetch(request *Request) {
	rep, req, err := c.Fetcher.Fetch(request, c.Spider)

	if err != nil {
		c.enqueueScrape(nil, err, request) // enqueue fetching error
		return
	}
	if rep != nil {
		rep.Request = request // tie request to response received
		c.Logger.WithFields(logrus.Fields{
			"event":   "RequestCrawled",
			"status":  rep.StatusCode,
			"request": request,
		}).Debugf("Crawled request %s, status %d", request, rep.StatusCode)
		ResponseReceived.Pub(c.Spider, request, rep)

		c.enqueueScrape(rep, nil, request) // enqueue fetching response
	} else { // fetcher can return request, i.e., redirect
		c.enqueueRequest(req)
	}
}

func (c *Crawler) enqueueScrape(response *Response, err error, request *Request) {
	c.WorkPool.SendWorkAsync(func() {
		c.scrape(response, err, request)
	}, nil)
}

func (c *Crawler) scrape(response *Response, err error, request *Request) {
	var result *SpiderResult
	if err == nil {
		// spider and spider middlewares scrape the response
		result, err = c.SpiderMiddlewareManager.ScrapeResponse(request, response, c.Spider)
	} else if request.Callback != nil {
		result, err = request.Callback(nil, err) // request callback handles fetching error
		if err != nil && err != ErrIgnoreRequest {
			c.Logger.WithError(err).Errorf("Fetching request %s", request)
		}
	}

	if err == nil {
		for _, req := range result.Requests {
			c.processSpiderRequest(req)
		}
		for _, item := range result.Items {
			c.processSpiderItem(item, response)
		}

	} else {
		if err == ErrSpiderClosed {
			return
		}
		c.Logger.WithError(err).Errorf("Processing request %s (referer: %s)", request, request.Header.Get("Referer"))
		SpiderError.Pub(c.Spider, response, err)
		c.Stats.Inc("SpiderError/" + reflect.TypeOf(err).Name())
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
				"event": "ItemDropped",
				"item":  item,
			}).Warnf("Dropped item %s", item)
			ItemDropped.Pub(c.Spider, response, item)
		} else if err != nil {
			c.Logger.WithError(err).Errorf("Processing item %s", item)
		} else {
			c.Logger.WithFields(logrus.Fields{
				"event": "ItemScraped",
				"item":  resultItem,
				"src":   response,
			}).Debugf("Scraped item %s from %s", resultItem, response)
		}
	}, nil)
}
