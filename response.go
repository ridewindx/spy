package spy

import (
	"io"
	"net/http"
	"mime"
	"fmt"
	"encoding/xml"
	"encoding/json"
	"io/ioutil"
	"golang.org/x/net/html/charset"
	"github.com/PuerkitoBio/goquery"
	"github.com/Workiva/go-datastructures/queue"
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

	htmlDoc *goquery.Document
}

func NewResponse(hr *http.Response) (r *Response, err error) {
	r = &Response{Response: hr}
	r.reader, err = charset.NewReader(r.Response.Body, r.ContentType())
	return
}

func (r *Response) ContentType() string {
	return r.Response.Header.Get("Content-Type")
}

func (r *Response) Close() {
	r.Response.Body.Close()
}

func (r *Response) Content(v interface{}) error {
	mediaType, _, err := mime.ParseMediaType(r.ContentType())
	if err != nil {
		return err
	}
	switch mediaType {
	case MIMEHTML:
		return r.HTMLDocument()
	default:
		return nil, fmt.Errorf("unknown media type: %s", mediaType)
	}
}

func (r *Response) HTMLDocument() (*goquery.Document, error) {
	var err error
	if r.htmlDoc == nil {
		r.htmlDoc, err = goquery.NewDocumentFromReader(r.reader)
	}
	return r.htmlDoc, err
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

func (r *Response) Selector() (Selector, error) {
	doc, err := r.HTMLDocument()
	if err != nil {
		return nil, err
	}

	return NewGoquerySelector(doc), nil
}

func (r *Response) Select(query string) (Selectors, error) {
	selector, err := r.Selector()
	if err != nil {
		return nil, err
	}

	return selector.Select(query), nil
}
