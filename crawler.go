package spy

import (
	"github.com/Sirupsen/logrus"
)

type Crawler struct {
	*Config
	*logrus.Logger

	Spider ISpider
	Scheduler IScheduler
	Fetcher IFetcher

	crawling bool
}

func NewCrawler(spider ISpider, scheduler IScheduler) *Crawler {
	return &Crawler{
		Spider: spider,
		Scheduler: scheduler,
	}
}

func (c *Crawler) Craw() {
	// TODO: signal handling for SIGTERM, SIGINT, SIGBREAK

	if c.crawling {
		panic("Crawling already taking place")
	}
	c.crawling = true


}

func (cr *Crawler) Stop() {
}

func (cr *Crawler) OpenSpider() {

}
