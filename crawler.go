package spy

import (
	"github.com/spf13/viper"
	"github.com/Sirupsen/logrus"
)

type Config struct {
	*viper.Viper
}

type CrawlerRunner struct {
	*Config
	*logrus.Logger

	crawlers []Crawler
}

func (cr *CrawlerRunner) Run() {
	// TODO: signal handling for SIGTERM, SIGINT, SIGBREAK
	//
}

func (cr *CrawlerRunner) Stop() {
	for _, c := range cr.crawlers {
		go c.stop()
	}
}

type Crawler struct {

}

func (c *Crawler) stop() {
}
