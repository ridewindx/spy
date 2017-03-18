package spy

import (
	"regexp"
	"github.com/PuerkitoBio/goquery"
)

type Selector interface {
	Select(query string) Selectors
	Regex(regex interface{}) []string
	Extract() string
	Attr(attrName string) (val string, exists bool)
}

type Selectors []Selector

func (ss Selectors) Select(query string) Selectors {
	var result Selectors
	for _, s := range ss {
		result = append(result, s.Select(query)...)
	}
	return result
}

func (ss Selectors) Regex(regex interface{}) []string {
	re := getRegex(regex)

	var result []string
	for _, s := range ss {
		result = append(result, s.Regex(re)...)
	}
	return result
}

func (ss Selectors) RegexFirst(regex interface{}) string {
	re := getRegex(regex)

	for _, s := range ss {
		for _, item := range s.Regex(re) {
			return item
		}
	}
	return ""
}

func (ss Selectors) Attrs(attrName string) []string {
	var result []string
	for _, s := range ss {
		val, exists := s.Attr(attrName)
		if exists {
			result = append(val, exists)
		}
	}
	return result
}

func (ss Selectors) Extract() []string {
	var result []string
	for _, s := range ss {
		result = append(result, s.Extract())
	}
	return result
}

func (ss Selectors) ExtractFirst() string {
	for _, s := range ss {
		return s.Extract()
	}
	return ""
}

type GoquerySelector struct {
	*goquery.Selection
}

func NewGoquerySelector(doc *goquery.Document) *GoquerySelector {
	return &GoquerySelector{
		Selection: doc.Selection,
	}
}

func (gs *GoquerySelector) Select(query string) Selectors {
	var result = make(Selectors, gs.Length())
	gs.Find(query).Each(func(i int, s *goquery.Selection) {
		result = append(result, Selector(GoquerySelector{s}))
	})
	return result
}

func (gs *GoquerySelector) Regex(regex interface{}) []string {
	re := getRegex(regex)

	var result []string
	for _, slice := range re.FindAllStringSubmatch(gs.Extract(), -1) {
		if len(slice) == 1 {
			result = append(result, slice[0]) // the whole match, no submatch
		} else {
			result = append(result, slice[1:]...) // submatches, exclude the whole match
		}
	}
	return result
}

func (gs *GoquerySelector) Extract() string {
	// TODO: html.UnescapeString
	return gs.Text()
}

func (gs *GoquerySelector) Attr(attrName string) (val string, exists bool) {
	return gs.Attr(attrName)
}

func getRegex(regex interface{}) *regexp.Regexp {
	switch re := regex.(type) {
	case *regexp.Regexp:
		return re
	case string:
		return regexp.MustCompile(re)
	default:
		panic("regex type must be either *regexp.Regexp or string")
	}
}
