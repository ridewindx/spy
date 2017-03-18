package spy

import (
	"fmt"
	"regexp"
	"sync"
	"strings"
)

// common file extensions that are not followed if they occur in links
var IgnoredExtensions = []string{
// images
"mng", "pct", "bmp", "gif ", "jpg", "jpeg", "png", "pst", "psp", "tif",
"tiff", "ai", "drw", "dxf", "eps", "ps", "svg",

// audio
"mp3", "wma", "ogg", "wav", "ra", "aac", "mid", "au", "aiff",

// video
"3gp", "asf", "asx", "avi", "mov", "mp4", "mpg", "qt", "rm", "swf", "wmv",
"m4a",

// office suites
"xls", "xlsx", "ppt", "pptx", "pps", "doc", "docx", "odt", "ods", "odg",
"odp",

// other
"css", "pdf", "exe", "bin", "rss", "zip", "rar",
}

var BaseUrlRe = regexp.MustCompile(`(?!)<base\s[^>]*href\s*=\s*[\"\']\s*([^\"\'\s]+)\s*[\"\']`)

// Link represents an extracted link.
type Link struct {
	url string
	text string
	fragment string
	noFollow bool
}

func (l *Link) String() string {
	return fmt.Sprintf("Link(url=%s, text=%s, fragment=%s, noFollow=%t", l.url, l.text, l.fragment, l.noFollow)
}

type LinkExtractor interface {
	ExtractLinks(response *Response) ([]Link, error)
}

type HTMLLinkExtractor struct {
	// Regular expressions that the (absolute) urls must match in order to be extracted.
	// If empty, it will match all links.
	Allows []string

	// regular expressions that the (absolute) urls must match in order to be excluded.
	// It has precedence over the Allows parameter.
	// If empty, it won't exclude any links.
	Denies []string

	// Domains which will be considered for extracting the links.
	AllowDomains []string

	// Domains which won't be considered for extracting the links.
	DenyDomains []string

	// File extensions that should be ignored when extracting links.
	// If empty, it will default to the IgnoredExtensions.
	DenyExtensions []string

	// Selectors of goquery(https://github.com/PuerkitoBio/goquery) which define
	// regions inside the response where links should be extracted from.
	// If given, only the text selected by those selectors will be scanned for links.
	RestrictSelectors []string

	// Whether duplicate filtering should be applied to extracted links.
	// Defaults to true.
	Unique bool

	// Function which receives each value extracted from the tag and attributes scanned
	// and can modify the value and return a new one, or return "" to ignore the link altogether.
	// If not given, defaults to the untouched link.
	ProcessValue func(value string) string

	// a list of tags to consider when extracting links.
	// Defaults to {"a", "area"}.
	Tags []string

	// Attributes which should be considered when looking for links to extract.
	// Only for those tags specified in the tags parameter.
	// Defaults to {"href"}.
	Attrs []string

	tags string
}

func (hle *HTMLLinkExtractor) Init() {
	if len(hle.Tags) == 0 {
		hle.Tags = []string{"a", "area"}
	}
	hle.tags = strings.Join(hle.Tags, ",")

	if len(hle.Attrs) == 0 {
		hle.Attrs = []string{"href"}
	}
}

func (hle *HTMLLinkExtractor) ExtractLinks(response *Response) ([]*Link, error) {
	baseUrl, err := response.getBaseUrl()
	if err != nil {
		return nil, err
	}

	var links []*Link
	for _, element := range response.Select(hle.tags) {
		for _, attrName := range hle.Attrs {
			attrVal, exists := element.Attr(attrName)
			if !exists {
				continue
			}

			attrVal = strings.Trim(attrVal, " \t\n\r\x0c") // strip html5 whitespaces
			attrVal = baseUrl.ResolveReference(attrVal)
			if attrVal == "" {
				continue
			}

			var noFollow bool
			rel, exists := element.Attr("rel")
			if exists {
				for _, s := range strings.Split(rel, " ") {
					if s == "nofollow" {
						noFollow = true
						break
					}
				}
			}

			links = append(links, &Link{
				url: attrVal,
				text: element.Extract(),
				noFollow: noFollow,
			})
		}
	}

	// TODO: unique
	return links, nil
}
