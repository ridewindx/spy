package spy

import "github.com/spf13/viper"

type Config struct {
	*viper.Viper

	ConcurrentRequests int
	ConcurrentRequestsPerDomain int
	ConcurrentRequestsPerIp int
	RandomizeFetchDelay bool
	FetchDelay float64
}

const ItemPipelines = "ItemPipelines"
