package spy

import (
	"github.com/spf13/viper"
	"github.com/Sirupsen/logrus"
)

type Config struct {
	*viper.Viper
}

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
