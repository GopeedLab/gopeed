package webview

type Provider interface {
	IsAvailable() bool
	Open(opts OpenOptions) (Page, error)
}

type unavailableProvider struct{}

func NewUnavailableProvider() Provider {
	return unavailableProvider{}
}

func (unavailableProvider) IsAvailable() bool {
	return false
}

func (unavailableProvider) Open(OpenOptions) (Page, error) {
	return nil, ErrUnavailable
}
