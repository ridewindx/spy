package spy

import "fmt"

type Opener interface {
	Open(spider ISpider)
}

type Closer interface {
	Close(spider ISpider)
}

func openAll(spider ISpider, openers ...interface{}) {
	for i := 0; i < len(openers); i++ {
		if opener, ok := openers[i].(Opener); ok {
			opener.Open(spider)
		}
	}
}

func closeAll(spider ISpider, closers ...interface{}) {
	for i := len(closers) - 1; i >= 0; i-- {
		if closer, ok := closers[i].(Closer); ok {
			closer.Close(spider)
		}
	}
}

func assert(value bool, formatAndArgs ...interface{}) {
	if !value {
		var msg string
		if len(formatAndArgs) == 1 {
			msg = formatAndArgs[0].(string)
		} else {
			msg = fmt.Sprintf(formatAndArgs[0].(string), formatAndArgs[1:]...)
		}
		panic(msg)
	}
}
