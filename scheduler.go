package spy

type IScheduler interface {
	Open(spider ISpider)
	Close()
	EnqueueRequest(request *Request)
	NextRequest() *Request
}

type Scheduler struct {

}

func NewScheduler() *Scheduler {
	return &Scheduler{

	}
}

func (s *Scheduler) Open(spider ISpider) {

}

func (s *Scheduler) Close() {

}

func (s *Scheduler) EnqueueRequest(request *Request) bool {

}

func (s *Scheduler) NextRequest() *Request {

}
