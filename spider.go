package spy

type Item struct {
	*Response
	item []string
}

type ISpider interface {
	startRequests() ([]*Request, error)
	parseNext(response *Response) (*Item, error)
}

type Spider struct {
	startURLs []string
}

func (s *Spider) startResusts() ([]*Request, error) {
	reqs := make([]*Request, 0, len(s.startURLs))
	for i, urlStr := range s.startURLs {
		req, err := NewRequest(urlStr, "")
		if err != nil {
			return nil, err
		}
		reqs[i] = req
	}
	return reqs, nil
}

func (s *Spider) parseNext(response *Response) (*Item, error) {
	panic("not implemented")
}

type CrawlSpider struct {
	*Spider
}

func (s *CrawlSpider) parseNext(response *Response) (*Item, error) {

}

type Rule struct {
	
}
