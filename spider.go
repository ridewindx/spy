package spy

type Spider struct {
	startRequests func() []Request
}

