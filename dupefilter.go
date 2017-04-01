package spy

type DupeFilter interface {
	Opener
	Closer
	SeenRequest(request *Request) bool
}

type FingerprintDupeFilter struct {

}
