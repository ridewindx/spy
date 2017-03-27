package spy

import (
	"github.com/Sirupsen/logrus"
)

type Crawler struct {
	*Config
	*logrus.Logger

	*Spider
}

func (c *Crawler) Run() {
	// TODO: signal handling for SIGTERM, SIGINT, SIGBREAK
}

func (cr *Crawler) Stop() {
}

func (cr *Crawler) openSpider() {

}
