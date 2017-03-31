package spy

import (
	"time"
	"math/rand"
	"fmt"
	"sync"
	"github.com/ridewindx/crumb/dnscache"
)

type IFetcher interface {
	Fetch(req *Request, spider ISpider) (*Response, *Request, error)
}

type Fetcher struct {
	Handlers          map[string]FetcherHandler
	TotalConcurrency  int
	DomainConcurrency int
	IpConcurrency     int
	Delay             time.Duration
	RandomizeDelay    bool
	*FetcherMiddlewareManager
	slots             map[string]*Slot
	dnscache          *dnscache.Resolver
	active            map[*Request]struct{}
	*rand.Rand
	mutex             *sync.RWMutex
	closed            chan struct{}
	waitGroup *sync.WaitGroup
}

type FetcherHandler interface {
	Fetch(request *Request, spider ISpider) (*Response, error)
	Close()
}

type fetchresult struct {
	rep *Response
	err error
}

type task struct {
	request *Request
	result chan *fetchresult
}

type Slot struct {
	concurrency int
	tasks       chan *task
	active      int
	lastSeen    time.Time
	closed chan struct{}
}

func NewFetcher() *Fetcher {
	return &Fetcher{
		Rand: rand.New(rand.NewSource(time.Now().UTC().UnixNano())),
	}
}

func (f *Fetcher) Open(spider ISpider) {

}

func (f *Fetcher) Close(spider ISpider) {
	for _, handler := range f.Handlers {
		handler.Close()
	}

	close(f.closed)
	for _, slot := range f.slots {
		close(slot.closed)
	}

	f.waitGroup.Wait()
}

func (f *Fetcher) Fetch(req *Request, spider ISpider) (*Response, *Request, error) {
	f.active[req] = struct{}{}
	defer delete(f.active, req)

	return f.FetcherMiddlewareManager.Fetch(f.fetchRequest, req, spider)
}

func (f *Fetcher) NeedsBackout() bool {
	return len(f.active) >= f.TotalConcurrency
}

func (f *Fetcher) fetchRequest(req *Request, spider ISpider) (*Response, error) {
	key := f.getSlotKey(req)
	f.mutex.RLock()
	slot, ok := f.slots[key]
	f.mutex.RUnlock()
	if !ok {
		slot = f.addSlot(key, spider)
	}

	result := make(chan *fetchresult)

	slot.tasks <- &task{req, result}

	r := <- result
	return r.rep, r.err
}

func (f *Fetcher) work(slot *Slot, spider ISpider) {
	f.waitGroup.Add(1)
	defer f.waitGroup.Done()

	for {
		select {
		case <-slot.closed:
			return

		default:
			now := time.Now()
			delay := f.computedDelay(spider)
			if delay > 0 {
				penalty := delay - now.Sub(slot.lastSeen)
				if penalty > 0 {
					time.Sleep(penalty)
					continue
				}
			}

			for len(slot.tasks) > 0 && (slot.concurrency-slot.active) > 0 {
				task := <-slot.tasks

				slot.lastSeen = time.Now()
				f.waitGroup.Add(1)
				go f.fetch(slot, task, spider)

				if delay > 0 {
					break
				}
			}
		}
	}
}

func (f *Fetcher) fetch(slot *Slot, task *task, spider ISpider) {
	defer f.waitGroup.Done()

	slot.active++
	defer func() {
		slot.active--
	}()

	req := task.request
	var rep *Response
	var err error
	scheme := req.Request.URL.Scheme // TODO: not only http url
	handler, ok := f.Handlers[scheme]
	if ok {
		rep, err = handler.Fetch(req, spider)
	} else {
		err = fmt.Errorf("unsupported URL scheme '%s'", scheme)
	}
	task.result <- &fetchresult{rep, err}
}

func (f *Fetcher) computedDelay(spider ISpider) time.Duration {
	var delay time.Duration
	if d := spider.FetchDelay(); d > 0 {
		delay = time.Duration(d)
	} else {
		delay = f.Delay
	}

	if f.RandomizeDelay {
		return time.Duration(0.5*delay + f.Rand.Int63n(int64(delay)))
	} else {
		return delay
	}
}

func (f *Fetcher) getSlotKey(req *Request) string {
	var key string
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
	return key
}

func (f *Fetcher) addSlot(key string, spider ISpider) *Slot {
	slot := &Slot{
		concurrency: spider.ConcurrentRequests(),
		closed: make(chan struct{}),
	}
	if slot.concurrency == 0 {
		if f.IpConcurrency > 0 {
			slot.concurrency = f.IpConcurrency
		} else {
			slot.concurrency = f.DomainConcurrency
		}
	}
	slot.tasks = make(chan *task, slot.concurrency)

	go f.work(slot, spider)

	f.mutex.Lock()
	f.slots[key] = slot
	f.mutex.Unlock()

	return slot
}

func (f *Fetcher) purgeSlots() {
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-f.closed:
			ticker.Stop()
			return

		case <-ticker.C:
			idleSlots := make(map[string]*Slot)
			f.mutex.RLock()
			for key, slot := range f.slots {
				if len(slot.tasks) == 0 && slot.active == 0 {
					idleSlots[key] = slot
				}
			}
			f.mutex.RUnlock()

			f.mutex.Lock()
			for key, slot := range idleSlots {
				delete(f.slots, key)
				close(slot.closed)
			}
			f.mutex.RUnlock()
		}
	}
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
