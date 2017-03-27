package spy

import (
	"github.com/ridewindx/crumb/concurrency"
	"reflect"
)

type OnSpiderOpeneder interface {
	OnSpiderOpened(spider ISpider)
}

type OnSpiderCloseder interface {
	OnSpiderClosed(spider Spider)
}

type FromCrawlerer interface {
	FromCrawler(crawler Crawler)
}

type FromConfiger interface {
	FromConfig(config Config)
}

type Middleware interface {
	OnSpiderOpeneder
	OnSpiderCloseder
}

type MiddlewareManager struct {
	middlewares []Middleware
	methods map[string][]interface{}
}

func NewMiddlewareManager() *MiddlewareManager {
	return &MiddlewareManager{
		methods: make(map[string][]interface{}),
	}
}

func (mm *MiddlewareManager) Register(middleware Middleware) {
	mm.middlewares = append(mm.middlewares, middleware)
	mm.methods["OnSpiderOpened"] = append(mm.methods["OnSpiderOpened"], middleware.OnSpiderOpened)
	mm.methods["OnSpiderClosed"] = append([]Middleware{middleware.OnSpiderClosed}, mm.methods["OnSpiderClosed"]...)
}

func (mm *MiddlewareManager) OnSpiderOpened(spider Spider) {
	for m := range mm.methods["OnSpiderOpened"] {
		m.(func(spider Spider))(spider)
	}
}

func (mm *MiddlewareManager) OnSpiderClosed(spider Spider) {
	for m := range mm.methods["OnSpiderClosed"] {
		m.(func(spider Spider))(spider)
	}
}

func (mm *MiddlewareManager) addHandler(middleware Middleware, handlerName string) {
	v := reflect.ValueOf(middleware)
	t := v.Type()
	m, ok := t.MethodByName(handlerName)
	if ok {
		mm.methods[handlerName] = append(mm.methods[handlerName], func(in []reflect.Value) reflect.Value {
			in = append([]reflect.Value{v}, in...)
			return m.Func.Call(in)
		})
	}
}

func (mm *MiddlewareManager) callHandler(handlerName string, object interface{}, args ...interface{}) interface{} {
	in := make([]reflect.Value, 1+len(args))
	in[0] = reflect.ValueOf(object)
	for i, arg := range args {
		in[i+1] = reflect.ValueOf(arg)
	}
	for _, m := range mm.methods(handlerName) {
		in[0] = m.(func([]reflect.Value) reflect.Value)(in)
	}
	return in[0].Interface()
}
