package spy

import (
	"fmt"
	"regexp"
	"strings"
	"net/url"
	"path"
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

// Link represents an extracted link.
type Link struct {
	url *url.URL
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

	// Selectors which define regions inside the response where links should be extracted from.
	// If given, only the text selected by those selectors will be scanned for links.
	RestrictSelectors []string

	// Whether duplicate filtering should be applied to extracted links.
	// Defaults to false.
	Unique bool

	// Function which receives each url value extracted from the tag and attributes scanned
	// and can modify the value and return a new one, or return nil to ignore the link altogether.
	// If not given, defaults to the untouched link.
	ProcessValue func(value *url.URL) *url.URL

	// a list of tags to consider when extracting links.
	// Defaults to {"a", "area"}.
	Tags []string

	// Attributes which should be considered when looking for links to extract.
	// Only for those tags specified in the tags parameter.
	// Defaults to {"href"}.
	Attrs []string

	allowRes []*regexp.Regexp
	denyRes []*regexp.Regexp
	tags string
}

func (hle *HTMLLinkExtractor) Init() {
	for _, allow := range hle.Allows {
		hle.allowRes = append(hle.allowRes, regexp.MustCompile(allow))
	}

	for _, deny := range hle.Denies {
		hle.denyRes = append(hle.denyRes, regexp.MustCompile(deny))
	}

	for i, domain := range hle.AllowDomains {
		if domain[0] != '.' {
			hle.AllowDomains[i] = "." + domain
		}
	}

	for i, domain := range hle.DenyDomains {
		if domain[0] != '.' {
			hle.DenyDomains[i] = "." + domain
		}
	}

	if len(hle.DenyExtensions) == 0 {
		hle.DenyExtensions = IgnoredExtensions
	} else {
		for i, ext := range hle.DenyExtensions {
			if ext[0] == '.' {
				hle.DenyExtensions[i] = ext[1:] // remove leading dot
			}
		}
	}

	if len(hle.Tags) == 0 {
		hle.Tags = []string{"a", "area"}
	}
	hle.tags = strings.Join(hle.Tags, ",")

	if len(hle.Attrs) == 0 {
		hle.Attrs = []string{"href"}
	}

	if hle.ProcessValue == nil {
		hle.ProcessValue = func(value *url.URL) *url.URL {
			return value
		}
	}
}

func (hle *HTMLLinkExtractor) ExtractLinks(response *Response) []*Link {
	baseUrl := response.getBaseUrl()

	var selectors Selectors
	if len(hle.RestrictSelectors) > 0 {
		for _, rs := range hle.RestrictSelectors {
			selectors = append(selectors, response.Select(rs))
		}
		selectors = selectors.Select(hle.tags)
	} else {
		selectors = response.Select(hle.tags)
	}

	var links []*Link
	for _, element := range selectors {
		for _, attrName := range hle.Attrs {
			attrVal, exists := element.Attr(attrName)
			if !exists {
				continue
			}

			attrVal = strings.Trim(attrVal, " \t\n\r\x0c") // strip html5 whitespaces
			u, err := url.Parse(attrVal)
			if err != nil {
				continue
			}
			u = baseUrl.ResolveReference(u)
			if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "file" {
				continue
			}
			u = hle.ProcessValue(u)
			if u == nil {
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
				url: u,
				text: element.Extract(),
				noFollow: noFollow,
			})
		}
	}

	var reducedLinks []*Link
	var urlset map[string]struct{}
	for _, link := range links {
		u := uniqueURL(link.url, true)
		if hle.Unique {
			_, ok := urlset[u]
			if ok {
				continue
			}
			urlset[u] = struct{}{}
		}

		if !urlMatch(u, hle.allowRes) {
			continue
		}

		if urlMatch(u, hle.denyRes) {
			continue
		}

		host := link.url.Hostname()

		if !hostFromDomains(host, hle.AllowDomains) {
			continue
		}

		if hostFromDomains(host, hle.DenyDomains) {
			continue
		}

		if denyExtension(link.url.Path, hle.DenyExtensions) {
			continue
		}

		reducedLinks = append(reducedLinks, link)
	}

	return reducedLinks
}

func urlMatch(url string, res []*regexp.Regexp) bool {
	for _, re := range res {
		if re.FindString(url) != "" {
			return true
		}
	}
	return false
}

func hostFromDomains(host string, domains []string) bool {
	for _, domain := range domains {
		if strings.HasSuffix(host, domain) {
			return true
		}
	}
	return false
}

func denyExtension(p string, extensions []string) bool {
	p = path.Ext(p)
	if p != "" {
		p = p[1:] // remove dot
	}

	for _, ext := range extensions {
		if p == ext {
			return true
		}
	}
	return false
}
