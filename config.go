package spy

import "github.com/spf13/viper"

type Config struct {
	*viper.Viper
}

const ItemPipelines = "ItemPipelines"
