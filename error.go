package spy

import "errors"

var (
	ErrSpiderClosed = errors.New("spider closed")
	ErrItemDropped = errors.New("item dropped")
	ErrIgnoreRequest = errors.New("request ignored")
)
