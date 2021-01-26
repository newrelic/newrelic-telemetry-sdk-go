package telemetry

type DataPoint struct {}

type RequestFactory interface {
	BuildRequest([]DataPoint, ...ClientOption)
}

type requestFactory struct {
	insertKey string
	noDefaultKey bool
	host string
	port uint
}

func (r requestFactory) BuildRequest(points []DataPoint, option ...ClientOption) {
	panic("implement me")
}

type ClientOption func(o *requestFactory)

func NewRequestFactory(options ...ClientOption) RequestFactory {
	f := &requestFactory{}
	for _, opt := range options {
		opt(f)
	}

	if f.insertKey == "" && !f.noDefaultKey {
		panic("Insert key option must be specified! (one of WithInsertKey or WithNoDefaultKey)")
	}

	return f
}

func WithInsertKey(insertKey string) ClientOption {
	return func(o *requestFactory) {
		o.insertKey = insertKey
	}
}

func WithNoDefaultKey() ClientOption {
	return func(o *requestFactory) {
		o.noDefaultKey = true
	}
}

func WithHost(host string) ClientOption {
	return func(o *requestFactory) {
		o.host = host
	}
}

func WithPort(port uint) ClientOption {
	return func(o *requestFactory) {
		o.port = port
	}
}