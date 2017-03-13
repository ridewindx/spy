package spy

import "net/http"

type Response struct {
	*http.Response
}
