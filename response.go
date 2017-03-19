package spy

import (
	"io"
	"net/http"
	"mime"
	"encoding/xml"
	"encoding/json"
	"io/ioutil"
	"golang.org/x/net/html/charset"
	"github.com/PuerkitoBio/goquery"
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
	reader    io.Reader

	MediaType string
	HTMLDoc   *goquery.Document
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
