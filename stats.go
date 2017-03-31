package spy

type Stats struct {
	Name          string
	uint64Buckets map[string]uint64
	stringBuckets map[string]string
}

func NewStats(name string) *Stats {
	return &Stats{
		Name:          name,
		uint64Buckets: make(map[string]uint64),
		stringBuckets: make(map[string]string),
	}
}

func (stats *Stats) Open(spider ISpider) {

}

func (stats *Stats) Close(spider ISpider) {

}

func (stats *Stats) Get(key string) uint64 {
	if v, ok := stats.stringBuckets[key]; ok {
		return v
	} else {
		return 0
	}
}

func (stats *Stats) Inc(key string) {
	if _, ok := stats.uint64Buckets[key]; ok {
		stats.uint64Buckets[key] += 1
	} else {
		stats.uint64Buckets[key] = 0
	}
	return
}

func (stats *Stats) Max(key string, value uint64) {
	if v, ok := stats.uint64Buckets[key]; ok {
		if value > v {
			stats.uint64Buckets[key] = value
		}
	} else {
		stats.uint64Buckets[key] = value
	}
}

func (stats *Stats) Min(key string, value uint64) {
	if v, ok := stats.uint64Buckets[key]; ok {
		if value < v {
			stats.uint64Buckets[key] = value
		}
	} else {
		stats.uint64Buckets[key] = value
	}
}

func (stats *Stats) SetStr(key, value string) {
	stats.stringBuckets[key] = value
}

func (stats *Stats) GetStr(key string) string {
	if v, ok := stats.stringBuckets[key]; ok {
		return v
	} else {
		return ""
	}
}

func (stats *Stats) Del(key string) {
	delete(stats.uint64Buckets[key], key)
	delete(stats.stringBuckets[key], key)
}

func (stats *Stats) Clear() {
	stats.uint64Buckets = make(map[string]uint64)
	stats.stringBuckets = make(map[string]string)
}
