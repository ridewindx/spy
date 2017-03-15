package spy

import "net/http"

type Request struct {
	*http.Request

	Meta map[string]interface{}
}

func NewRequest(urlStr, method string) (*Request, error) {
	req, err := http.NewRequest(method, urlStr, nil)
	if err != nil {
		return nil, err
	}

	return &Request{
		Request: req,
		Meta: make(map[string]interface{}),
	}, nil
}
