package spy

import (
	"time"
	"math/rand"
	"fmt"
	"github.com/Workiva/go-datastructures/queue"
	"github.com/ridewindx/crumb/dnscache"
)

type IFetcher interface {
	Fetch(req *Request, spider ISpider) (*FetchResult, error)
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
	Fetch(request *Request, spider ISpider) *Response
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

type FetchResult struct {
	*Request
	*Response
	error
}

func (f *Fetcher) Fetch(req *Request, spider ISpider) (*FetchResult, error) {
	f.active[req] = struct{}{}
	defer delete(f.active, req)

	// TODO
	result := f.FetcherMiddlewareManager.Fetch(f.enqueueRequest, req, spider)
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

func (fmr *FetchResult) Empty() bool {
	return fmr == nil || (fmr.Request == nil && fmr.Response == nil && fmr.error == nil)
}

type FetcherMiddleware interface {
	Middleware

	// One or more of the following methods should be implemented.
	// ProcessRequest(request *Request, spider ISpider) *FetchResult
	// ProcessResponse(response *FetchResult, spider ISpider) *FetchResult
	// ProcessException(exception *FetchResult, spider ISpider) *FetchResult
}

type FetcherMiddlewareManager struct {
	*MiddlewareManager
}

func NewFetcherMiddlewareManager() *FetcherMiddlewareManager {
	return &FetcherMiddlewareManager{
		MiddlewareManager: NewMiddlewareManager(),
	}
}

func (fmm *FetcherMiddlewareManager) Register(middleware FetcherMiddleware) {
	mw := Middleware(middleware)
	fmm.MiddlewareManager.Register(mw)
	fmm.addHandler(mw, "ProcessRequest")
	fmm.addHandler(mw, "ProcessResponse", true)
	fmm.addHandler(mw, "ProcessException", true)
}

func (fmm *FetcherMiddlewareManager) Fetch(fetcherFunc func(*Request, ISpider) *FetchResult, req *Request, spider ISpider) *FetchResult {
	var result *FetchResult

	iter := fmm.callHandlerIterator("ProcessRequest", req, spider)
	for iter.HasNext() {
		result = iter.Next().(*FetchResult)
		if !result.Empty() {
			break
		}
	}

	if result.Empty() {
		result = fetcherFunc(req, spider)
	}

	if result.error != nil {
		iter = fmm.callHandlerIterator("ProcessException", result, spider)
		for iter.HasNext() {
			result = iter.Next().(*FetchResult)
			if !result.Empty() {
				break
			}
		}
	}

	if result.error == nil {
		iter = fmm.callHandlerIterator("ProcessResponse", result, spider)
		for iter.HasNext() {
			result = iter.Next().(*FetchResult)
			if !result.Empty() {
				break
			}
			if result.Request != nil {
				return result
			}
		}
	}
	return result
}
