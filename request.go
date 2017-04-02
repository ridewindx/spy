package spy

import (
	"bytes"
	"net/http"
	"net/url"
	"sort"
	"github.com/ridewindx/crumb/weakref"
	"crypto/sha1"
	"bufio"
	"encoding/hex"
)

type Request struct {
	*http.Request
	Error     error
	Meta      map[string]interface{}
	NotFilter bool
	Callback  func(response *Response, err error) (*SpiderResult, error)
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

var fingerprintCache = weakref.NewWeakPtrMap()

// Fingerprint returns a hash that uniquely identifies the request.
// Ignore all headers.
func (req *Request) Fingerprint() string {
	val, ok := fingerprintCache.Get(req)
	if ok {
		return val.(string)
	} else {
		h := sha1.New()

		buf := bufio.NewWriterSize(h, 1024)
		buf.WriteString(req.Method)
		buf.WriteString(uniqueURL(req.URL, false))
		body, err := req.GetBody()
		if err != nil {
			panic("request.GetBody returns error: "+err.Error())
		}
		buf.ReadFrom(body)
		buf.Flush()

		fingerprint := hex.EncodeToString(h.Sum(nil))
		fingerprintCache.Put(req, fingerprint)

		return fingerprint
	}
}

// uniqueURL computes a string from the url by:
// - retain query arguments with blank values
// - sort query arguments, first by key, then by value
// - escape query string
// - remove fragments
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
			buf.WriteString(url.QueryEscape(k))
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
		}
	}
	buf.Bytes()[mark] = '?'
	return buf.String()
}
