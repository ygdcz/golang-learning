package http_learning

import "net/http"

type OurCustomTransport struct {
	BaseTransport http.RoundTripper
}

func (t *OurCustomTransport) transport() http.RoundTripper {
	if t.BaseTransport != nil {
		return t.BaseTransport
	}
	return http.DefaultTransport
}

func (t *OurCustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.transport().RoundTrip(req)
}

func (t *OurCustomTransport) Client() *http.Client {
	return &http.Client{Transport: t}
}

func Customtrans() {
	t := &OurCustomTransport{}

	c := t.Client()
	c.Get("")
}
