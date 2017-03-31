package spy

type ItemPipelineMiddleware interface{}

type ItemProcessor interface {
	ProcessItem(item *Item, spider ISpider) (*Item, error)
}

type ItemPipelineManager struct {
	middlewares    []ItemPipelineMiddleware
	itemProcessors []ItemProcessor
}

func (ipm *ItemPipelineManager) Register(middleware ItemPipelineMiddleware) {
	ipm.middlewares = append(ipm.middlewares, middleware)

	if processor, ok := middleware.(ItemProcessor); ok {
		ipm.itemProcessors = append(ipm.itemProcessors, processor)
	}
}

func (ipm *ItemPipelineManager) Open(spider ISpider) {
	openAll(spider, ipm.middlewares...)
}

func (ipm *ItemPipelineManager) Close(spider ISpider) {
	closeAll(spider, ipm.middlewares...)
}

func (ipm *ItemPipelineManager) ProcessItem(item *Item, spider ISpider) (*Item, error) {
	var err error
	for _, processor := range ipm.itemProcessors {
		item, err = processor.ProcessItem(item, spider)
		if err != nil {
			return nil, err
		}
	}
	return item, nil
}
