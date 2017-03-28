package spy

type ItemPipelineManager struct {
	*MiddlewareManager
}

func NewItemPipelineManager() *ItemPipelineManager {
	return &ItemPipelineManager{
		MiddlewareManager: NewMiddlewareManager(),
	}
}

func (ipm *ItemPipelineManager) Register(middleware Middleware) {
	ipm.MiddlewareManager.Register(middleware)
	ipm.addHandler(middleware, "ProcessItem")
}

func (ipm *ItemPipelineManager) ProcessItem(item *Item, spider ISpider) *Item {
	return ipm.callHandler("ProcessItem", item, spider).(*Item)
}
