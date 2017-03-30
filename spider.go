package spy

type Item map[string]interface{}

type SpiderResult struct {
	Requests []*Request
	Items []*Item
}

func (sr *SpiderResult) Empty() bool {
	return sr == nil || (len(sr.Requests) == 0 && len(sr.Items) == 0)
}

type ISpider interface {
	StartRequests() []*Request
	Parse(response *Response) (*SpiderResult, error)
	String() string
}

type Spider struct {
	Name      string
	StartURLs []string
}

func (s *Spider) StartResusts() []*Request {
	reqs := make([]*Request, len(s.StartURLs))
	for i, url := range s.StartURLs {
		reqs[i] = NewRequest(url, "")
	}
	return reqs
}

func (s *Spider) Parse(response *Response) (*SpiderResult, error) {
	panic("not implemented")
}

func (s *Spider) String() string {
	return s.Name
}

type CrawlSpider struct {
	*Spider
}

func (s *CrawlSpider) Parse(response *Response) (*SpiderResult, error) {

}

type Rule struct {
	
}

type SpiderMiddleware interface {}

type SpiderInputProcessor interface {
	ProcessSpiderInput(response *Response, spider ISpider) error
}

type SpiderOutputProcessor interface {
	ProcessSpiderOutput(result *SpiderResult, response *Response, spider ISpider) (*SpiderResult, error)
}

type SpiderErrorProcessor interface {
	ProcessSpiderError(err error, response *Response, spider ISpider) (*SpiderResult, error)
}

type StartRequestsProcessor interface {
	ProcessStartRequests(startRequests []*Request, spider ISpider) ([]*Request, error)
}

type SpiderMiddlewareManager struct {
	middlewares []SpiderMiddleware
	spiderInputProcessors []SpiderInputProcessor
	spiderOutputProcessors []SpiderOutputProcessor
	spiderErrorProcessors []SpiderErrorProcessor
	startRequestsProcessors []StartRequestsProcessor
}

func (smm *SpiderMiddlewareManager) Register(middleware SpiderMiddleware) {
	smm.middlewares = append(smm.middlewares, middleware)

	if p, ok := middleware.(SpiderInputProcessor); ok {
		smm.spiderInputProcessors = append(smm.spiderInputProcessors, p)
	}
	if p, ok := middleware.(SpiderOutputProcessor); ok {
		smm.spiderOutputProcessors = append([]SpiderOutputProcessor{p}, smm.spiderOutputProcessors...)
	}
	if p, ok := middleware.(SpiderErrorProcessor); ok {
		smm.spiderErrorProcessors = append([]SpiderErrorProcessor{p}, smm.spiderErrorProcessors...)
	}
	if p, ok := middleware.(StartRequestsProcessor); ok {
		smm.startRequestsProcessors = append([]StartRequestsProcessor{p}, smm.startRequestsProcessors...)
	}
}

func (smm *SpiderMiddlewareManager) ScrapeResponse(request *Request, response *Response, spider ISpider) (*SpiderResult, error) {
	var result *SpiderResult
	var err error

	for _, processor := range smm.spiderInputProcessors {
		err = processor.ProcessSpiderInput(response, spider)
		if err != nil {
			break
		}
	}

	if err == nil {
		if request.Callback != nil {
			result, err = request.Callback(response, nil) // request callback handles response
		} else {
			result, err = spider.Parse(response) // spider parse response
		}
	} else if request.Callback != nil {
		result, err = request.Callback(nil, err) // request callback handles error
	}

	if err != nil {
		for _, processor := range smm.spiderErrorProcessors {
			result, err = processor.ProcessSpiderError(err, response, spider)
			if err != nil || result != nil {
				break
			}
		}
	}

	if err == nil && result != nil {
		for _, processor := range smm.spiderOutputProcessors {
			result, err = processor.ProcessSpiderOutput(result, response, spider)
			if err != nil {
				break
			}
		}
	}

	return result, err
}

func (smm *SpiderMiddlewareManager) ProcessStartRequests(startRequests []*Request, spider ISpider) ([]*Request, error) {
	var result = startRequests
	var err error
	for _, processor := range smm.startRequestsProcessors {
		result, err = processor.ProcessStartRequests(startRequests, spider)
		if err != nil {
			return nil, err
		}
	}
	return result, err
}
