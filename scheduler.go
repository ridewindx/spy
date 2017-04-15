package spy

type IScheduler interface {
	Opener
	Closer
	EnqueueRequest(request *Request) bool
	NextRequest() *Request
}

type Scheduler struct {
	dupeFilter DupeFilter
}

func NewScheduler() *Scheduler {
	return &Scheduler{}
}

func (s *Scheduler) Open(spider ISpider) {
	s.dupeFilter.Open(spider)
}

func (s *Scheduler) Close(spider ISpider) {
	s.dupeFilter.Close(spider)
}

func (s *Scheduler) EnqueueRequest(request *Request) bool {
	if !request.NotFilter && s.dupeFilter.SeenRequest(request) {
		return false
	}



}

func (s *Scheduler) NextRequest() *Request {

}
