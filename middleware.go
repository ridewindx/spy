package spy

import (
	"github.com/ridewindx/crumb/concurrency"
	"reflect"
)

type OnSpiderOpeneder interface {
	OnSpiderOpened(spider ISpider)
}

type OnSpiderCloseder interface {
	OnSpiderClosed(spider ISpider)
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
	mm.methods["OnSpiderClosed"] = append([]interface{}{middleware.OnSpiderClosed}, mm.methods["OnSpiderClosed"]...)
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

func (mm *MiddlewareManager) addHandler(middleware Middleware, handlerName string, prepend ...bool) {
	v := reflect.ValueOf(middleware)
	t := v.Type()
	m, ok := t.MethodByName(handlerName)
	if ok {
		f := func(in []reflect.Value) reflect.Value {
			in = append([]reflect.Value{v}, in...)
			return m.Func.Call(in)
		}
		if len(prepend) == 0 || !prepend[0] {
			mm.methods[handlerName] = append(mm.methods[handlerName], f)
		} else {
			mm.methods[handlerName] = append([]interface{}{f}, mm.methods[handlerName]...)
		}
	}
}

func (mm *MiddlewareManager) callHandler(handlerName string, object interface{}, args ...interface{}) interface{} {
	iter := mm.callHandlerIterator(handlerName, object, args...)
	var result interface{}
	for iter.HasNext() {
		result = iter.Next()
	}
	return result
}

func (mm *MiddlewareManager) callHandlerIterator(handlerName string, object interface{}, args ...interface{}) *MiddlewareManagerIterator {
	in := make([]reflect.Value, 1+len(args))
	in[0] = reflect.ValueOf(object)
	for i, arg := range args {
		in[i+1] = reflect.ValueOf(arg)
	}
	return &MiddlewareManagerIterator{
		args:    in,
		methods: mm.methods(handlerName),
	}
}

type MiddlewareManagerIterator struct {
	args    []reflect.Value
	methods []interface{}
	pos     int
}

func (mmi *MiddlewareManagerIterator) HasNext() bool {
	return pos < len(in)
}

func (mmi *MiddlewareManagerIterator) Next() interface{} {
	mmi.args[0] = mmi.methods[pos].(func([]reflect.Value) reflect.Value)(mmi.args)
	pos++
	return mmi.args[0].Interface()
}
