package spy

import (
	"net/http"
	"net/url"
	"sort"
	"bytes"
)

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
