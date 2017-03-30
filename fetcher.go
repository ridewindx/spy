package spy

import (
	"time"
	"math/rand"
	"fmt"
	"github.com/Workiva/go-datastructures/queue"
	"github.com/ridewindx/crumb/dnscache"
)

type IFetcher interface {
	Fetch(req *Request, spider ISpider) (*Response, *Request, error)
}

type Fetcher struct {
	Handlers map[string]FetcherHandler
	TotalConcurrency int
	DomainConcurrency int
	IpConcurrency int
	Delay time.Duration
	RandomizeDelay bool
	*FetcherMiddlewareManager
	slots map[string]*Slot
	dnscache *dnscache.Resolver
	active map[*Request]struct{}
	*rand.Rand
}

type FetcherHandler interface {
	Fetch(request *Request, spider ISpider) (*Response, error)
	Close()
}

type Slot struct {
	concurrency int
	queue *queue.Queue
	active map[*Request]struct{}
	transferring map[*Request]struct{}
	lastSeen time.Time
}

func (s *Slot) freeSlots() int {
	return s.concurrency - len(s.transferring)
}

func NewFetcher() *Fetcher {
	return &Fetcher{
		Rand: rand.New(rand.NewSource(time.Now().UTC().UnixNano())),
	}
}

func (f *Fetcher) Fetch(req *Request, spider ISpider) (*FetchResult, error) {
	f.active[req] = struct{}{}
	defer delete(f.active, req)

	rep, req, err := f.FetcherMiddlewareManager.Fetch(f.enqueueRequest, req, spider)
	if result.error != nil {
		return nil, result.error
	}
	return result, nil
}

func (f *Fetcher) NeedsBackout() bool {
	return len(f.active) >= f.TotalConcurrency
}

func (f *Fetcher) Close() {
	for _, handler := range f.Handlers {
		handler.Close()
	}
}

func (f *Fetcher) enqueueRequest(req *Request, spider ISpider) {
	key, slot := f.getSlot(req)

	slot.active[req] = struct{}{}
	defer delete(slot.active, req)

	err := slot.queue.Put(req)
	if err != nil {
		panic(err)
	}
	f.processQueue(slot, spider)
}

func (f *Fetcher) processQueue(slot *Slot, spider ISpider) {
	now := time.Now()
	delay := f.computedDelay(spider)
	if delay > 0 {
		penalty := delay - now.Sub(slot.lastSeen)
		if penalty > 0 {
			// TODO:
			return
		}
	}

	for !slot.queue.Empty() && slot.freeSlots() > 0 {
		slot.lastSeen = now
		items, err := slot.queue.Poll(1, 1)
		if err != nil {
			panic(err) // TODO:
		}
		req := items[0].(*Request)
		if delay > 0 {
			f.processQueue(slot, spider)
			break
		}
	}
	return
}

func (f *Fetcher) fetch(slot *Slot, req *Request, spider ISpider) (*Response, error){
	slot.transferring[req] = struct{}{}
	defer delete(slot.transferring, req)

	f.processQueue(slot, spider) // TODO:

	scheme := req.Request.URL.Scheme // TODO: not only http url
	handler, ok := f.Handlers[scheme]
	if !ok {
		return nil, fmt.Errorf("unsupported URL scheme '%s'", scheme)
	}
	return handler.Fetch(req, spider), nil
}

func (f *Fetcher) computedDelay(spider ISpider) time.Duration {
	if f.RandomizeDelay {
		return time.Duration(0.5*f.Delay + f.Rand.Int63n(int64(f.Delay)))
	} else {
		return f.Delay
	}
}

func (f *Fetcher) getSlot(req *Request) (key string, slot *Slot) {
	if k, ok := req.Meta["downloadSlot"]; ok {
		key = k.(string)
	} else {
		key = req.URL.Host // TODO: strip port
		if f.IpConcurrency {
			k, err := f.dnscache.FetchOneString(key)
			if err == nil {
				key = k
			}
		}
		req.Meta["downloadSlot"] = key
	}

	slot, ok := f.slots[key]
	if !ok {
		slot = &Slot{
			queue: queue.New(8), // TODO: queue size hint
		}
		if f.IpConcurrency {
			slot.concurrency = f.IpConcurrency
		} else {
			slot.concurrency = f.DomainConcurrency
		}
		f.slots[key] = slot
	}
	return
}

type FetcherMiddleware interface {}

type FetchingRequestProcessor interface {
	/* ProcessRequest is called for each request that goes through the fetcher middleware.
	It should either: return nils, return a Response object, return a Request object, or return an IgnoreRequest error.

	If it returns nils, will continue processing this request, executing all other middlewares,
	until, finally, the appropriate fetcher handler performs the request performed (and its response downloaded).

	If it returns a Response object, won’t bother calling any other ProcessRequest or ProcessError methods,
	or the appropriate fetcher handler; it’ll return that response. The ProcessResponse methods of installed middleware
	is always called on every response.

	If it returns a Request object, will stop calling ProcessRequest methods and reschedule the returned request.
	Once the newly returned request is performed, the appropriate middleware chain will again be called on the downloaded response.

	If it returns an IgnoreRequest error, the ProcessError methods of installed downloader middleware will be called.
	If none of them handle the error, the callback function of the request (Request.Callback) is called.
	If no code handles the returned error, it is ignored and not logged (unlike other error).
	 */
	ProcessRequest(request *Request, spider ISpider) (*Response, *Request, error)
}

type FetchingResponseProcessor interface {
	/* ProcessResponse should either: return a Response object, return a Request object or return an IgnoreRequest error.

	If it returns a Response (it could be the same given response, or a brand-new one), that response will continue to be
	processed with the ProcessResponse method of the next middleware in the chain.

	If it returns a Request object, the middleware chain is halted and the returned request is rescheduled to be performed
	in the future. This is the same behavior as if a request is returned from ProcessRequest.

	If it returns an IgnoreRequest error, the callback function of the request (Request.Callback) is called.
	If no code handles the returned error, it is ignored and not logged (unlike other error).
	 */
	ProcessResponse(response *Response, request *Request, spider ISpider) (*Response, *Request, error)
}

type FetchingErrorProcessor interface {
	/* ProcessError is called when a fetcher handler or a ProcessRequest (from a fetcher middleware) returns an error.
	It should return: either nils, a Response object, or a Request object.

	If it returns nils, will continue processing this error, executing any other ProcessError methods of installed
	middleware, until no middleware is left.

	If it returns a Response object, the ProcessResponse methods chain of installed middleware is started,
	and won’t bother calling any other ProcessError methods of middleware.

	If it returns a Request object, the returned request is rescheduled to be performed in the future.
	This stops the execution of ProcessError methods of the middleware the same as returning a response would.
	 */
	ProcessError(err error, request *Request, spider ISpider) (*Response, *Request)
}

type FetcherMiddlewareManager struct {
	middlewares []FetcherMiddleware
	requestProcessors []FetchingRequestProcessor
	responseProcessors []FetchingResponseProcessor
	errorProcessors []FetchingErrorProcessor
}

func (fmm *FetcherMiddlewareManager) Register(middleware FetcherMiddleware) {
	fmm.middlewares = append(fmm.middlewares, middleware)

	if p, ok := middleware.(FetchingRequestProcessor); ok {
		fmm.requestProcessors = append(fmm.requestProcessors, p)
	}
	if p, ok := middleware.(FetchingResponseProcessor); ok {
		fmm.responseProcessors = append([]FetchingResponseProcessor{p}, fmm.responseProcessors...)
	}
	if p, ok := middleware.(FetchingErrorProcessor); ok {
		fmm.errorProcessors = append([]FetchingErrorProcessor{p}, fmm.errorProcessors...)
	}
}

func (fmm *FetcherMiddlewareManager) Fetch(fetchFunc func(*Request, ISpider) (*Response, error), request *Request, spider ISpider) (*Response, *Request, error) {
	var rep *Response
	var req *Request
	var err error

	var none = true
	for _, processor := range fmm.requestProcessors { // chain handles request
		rep, req, err = processor.ProcessRequest(request, spider)
		if rep != nil || req != nil || err != nil {
			none = false
			break
		}
	}

	if none {
		rep, err = fetchFunc(request, spider) // really fetch the request
		assert(rep != nil || err != nil)
	}

	if err != nil {
		for _, processor := range fmm.errorProcessors { // chain handles error
			rep, req = processor.ProcessError(err, request, spider)
			if rep != nil || req != nil {
				err = nil // ProcessError eliminated error
				break
			}
		}
	}

	assert(rep != nil || req != nil || err != nil)

	if err != nil {
		return nil, nil, err
	}

	if req != nil {
		return nil, req, nil // reschedule the request
	}

	for _, processor := range fmm.responseProcessors { // chain handles response
		rep, req, err = processor.ProcessResponse(rep, request, spider)
		if req != nil {
			return nil, req, nil // reschedule the request
		}
		if err != nil {
			return nil, nil, err
		}
		assert(rep != nil)
	}

	return rep, nil, nil
}
