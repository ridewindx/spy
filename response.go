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
	Reader io.Reader
}

func NewResponse(hr *http.Response) (r *Response, err error) {
	r = &Response{Response: hr}
	r.Reader, err = charset.NewReader(r.Response.Body, r.ContentType())
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
	return goquery.NewDocumentFromReader(r.Reader)
}

func (r *Response) Text() (text string, err error) {
	bytes, err := ioutil.ReadAll(r.Reader)
	if err == nil {
		text = string(bytes)
	}
	return
}

func (r *Response) XML(v interface{}) error {
	return xml.NewDecoder(r.Reader).Decode(v)
}

func (r *Response) JSON(v interface{}) error {
	return json.NewDecoder(r.Reader).Decode(v)
}
