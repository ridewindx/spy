package spy

type ItemPipelineMiddleware interface{}

type ItemProcessor interface {
	ProcessItem(item *Item, spider ISpider) (*Item, error)
}

type ItemPipelineManager struct {
	middlewares []ItemPipelineMiddleware
	itemProcessors []ItemProcessor
}

func (ipm *ItemPipelineManager) Register(middleware ItemPipelineMiddleware) {
	ipm.middlewares = append(ipm.middlewares, middleware)

	if p, ok := middleware.(ItemProcessor); ok {
		ipm.itemProcessors = append(ipm.itemProcessors, p)
	}
}

func (ipm *ItemPipelineManager) ProcessItem(item *Item, spider ISpider) (*Item, error) {
	for _, processor := range ipm.itemProcessors {
		item, err := processor.ProcessItem(item, spider)
		if err != nil {
			return nil, err
		}
	}
	return item, nil
}
