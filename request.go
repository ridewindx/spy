package spy

import "net/http"

type Request struct {
	*http.Request

	Meta map[string]interface{}
}
