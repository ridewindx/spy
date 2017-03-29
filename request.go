package spy

import (
	"net/http"
	"net/url"
	"sort"
	"bytes"
)

type Request struct {
	*http.Request
	Error error
	Meta map[string]interface{}
	NotFilter bool
	Callback func(response *Response, err error) (*SpiderResult, error)
}

func NewRequest(urlStr, method string) *Request {
	var req = &Request{
		Meta: make(map[string]interface{}),
	}
	// TODO: safe_url_string
	// TODO: escape_ajax
	req.Request, req.Error = http.NewRequest(method, urlStr, nil)
	return req
}

func uniqueURL(u *url.URL, ignoreQuery bool) string {
	forceQuery := u.ForceQuery
	rawQuery := u.RawQuery
	u.ForceQuery = false
	u.RawQuery = ""
	buf := bytes.NewBufferString(u.RequestURI())
	u.ForceQuery = forceQuery
	u.RawQuery = rawQuery

	if ignoreQuery {
		return buf.String()
	}

	queries, err := url.ParseQuery(rawQuery)
	if err != nil || len(queries) == 0 {
		return buf.String()
	}

	mark := buf.Len()

	ks := make([]string, 0, len(queries))
	for k := range queries {
		ks = append(ks, k)
	}
	sort.Strings(ks)

	for _, k := range ks {
		vs := queries[k]
		sort.Strings(vs)
		for _, v := range vs {
			buf.WriteByte('&')
			buf.WriteString(k)
			buf.WriteByte('=')
			buf.WriteString(v)
		}
	}
	buf.Bytes()[mark] = '?'
	return buf.String()
}
