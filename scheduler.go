package spy

type IScheduler interface {
	Opener
	Closer
	EnqueueRequest(request *Request) bool
	NextRequest() *Request
}

type Scheduler struct {
}

func NewScheduler() *Scheduler {
	return &Scheduler{}
}

func (s *Scheduler) Open(spider ISpider) {

}

func (s *Scheduler) Close(spider ISpider) {

}

func (s *Scheduler) EnqueueRequest(request *Request) bool {

}

func (s *Scheduler) NextRequest() *Request {

}
