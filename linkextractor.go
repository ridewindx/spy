package spy

import "fmt"

// common file extensions that are not followed if they occur in links
var ignoredExtensions = []string{
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
	url string
	text string
	fragment string
	noFollow bool
}

func (l *Link) String() string {
	return fmt.Sprintf("Link(url=%s, text=%s, fragment=%s, noFollow=%t", l.url, l.text, l.fragment, l.noFollow)
}

type LinkExtrator interface {
	extractLinks(response *Response) ([]Link, error)
}
