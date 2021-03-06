package spy

import (
	"encoding/json"
	"encoding/xml"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
)

const (
	MIMEJSON              = "application/json"
	MIMEHTML              = "text/html"
	MIMEXML               = "application/xml"
	MIMEXMLText           = "text/xml"
	MIMEPlain             = "text/plain"
	MIMEPOSTForm          = "application/x-www-form-urlencoded"
	MIMEMultipartPOSTForm = "multipart/form-data"
	MIMEPROTOBUF          = "application/x-protobuf"
)

type Response struct {
	*http.Response
	reader io.Reader

	MediaType string
	HTMLDoc   *goquery.Document

	/* Request which generated this response.
		This attribute is assigned in the `Crawler`, after the response and the request have passed
	    through all `Fetcher Middlewares`. In particular, this means that:

	    - HTTP redirections will cause the original request (to the URL before
	      redirection) to be assigned to the redirected response (with the final
	      URL after redirection).

	    - Response.Request.URL doesn't always equal Response.Response.URL

	    - This attribute is only available in the spider code, and in the `Spider Middlewares`,
	      but not in `Downloader Middlewares` (although you have the Request available there by
	      other means) and handlers of the `response_downloaded` signal.
	*/
	*Request
}

func NewResponse(hr *http.Response) (r *Response, err error) {
	r = &Response{Response: hr}

	r.MediaType, _, err = mime.ParseMediaType(r.ContentType())
	if err != nil {
		return
	}

	r.reader, err = charset.NewReader(r.Response.Body, r.ContentType())
	if err != nil {
		return
	}

	if r.MediaType == MIMEHTML {
		r.HTMLDoc, err = goquery.NewDocumentFromReader(r.reader)
	}

	return
}

func (r *Response) ContentType() string {
	return r.Response.Header.Get("Content-Type")
}

func (r *Response) Close() {
	r.Response.Body.Close()
}

func (r *Response) Text() (text string, err error) {
	bytes, err := ioutil.ReadAll(r.reader)
	if err == nil {
		text = string(bytes)
	}
	return
}

func (r *Response) XML(v interface{}) error {
	return xml.NewDecoder(r.reader).Decode(v)
}

func (r *Response) JSON(v interface{}) error {
	return json.NewDecoder(r.reader).Decode(v)
}

func (r *Response) Selector() Selector {
	return NewGoquerySelector(r.HTMLDoc)
}

func (r *Response) Select(query string) Selectors {
	return r.Selector().Select(query)
}

func (r *Response) getBaseUrl() *url.URL {
	bases := r.Select("html > head > base").Attrs("href")
	if len(bases) > 0 {
		baseUrl, err := url.Parse(bases[0])
		if err == nil {
			return r.Response.Request.URL.ResolveReference(baseUrl)
		}
	}
	return r.Response.Request.URL
}
