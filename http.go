package binder

import (
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

type HttpBindableRequest struct {
	*http.Request
}

func (r HttpBindableRequest) GetBody() io.Reader {
	return r.Body
}

func (r HttpBindableRequest) GetPathPattern() string {
	return r.Pattern
}

func (r HttpBindableRequest) GetPathValue(key string) string {
	return r.PathValue(key)
}

func (r HttpBindableRequest) GetQuery() url.Values {
	return r.URL.Query()
}

func (r HttpBindableRequest) headersToValues(headers http.Header) url.Values {
	values := url.Values{}
	for key, val := range headers {
		for _, v := range val {
			values.Add(key, v)
		}
	}
	return values
}

func (r HttpBindableRequest) GetHeaders() url.Values {
	return r.headersToValues(r.Header)
}

func (r HttpBindableRequest) GetContentLength() int64 {
	return r.ContentLength
}

func (r HttpBindableRequest) GetContentType() string {
	return r.Header.Get("Content-Type")
}

func (r HttpBindableRequest) GetForm() (url.Values, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	return r.Form, nil
}

func (r HttpBindableRequest) GetMultipartForm(maxBodySize int64) (*multipart.Form, error) {
	return r.MultipartForm, r.ParseMultipartForm(maxBodySize)
}

func NewHttpBindableRequest(r *http.Request) HttpBindableRequest {
	return HttpBindableRequest{r}
}

// BindHttp binds an http.Request to a struct or map.
func BindHttp(r *http.Request, i interface{}) error {
	return Bind(NewHttpBindableRequest(r), i)
}

// BindHttpBody binds an http.Request body to a struct or map.
func BindHttpBody(r *http.Request, i interface{}) error {
	return BindBody(NewHttpBindableRequest(r), i)
}

func BindHttpPath(r *http.Request, i interface{}) error {
	return BindPath(NewHttpBindableRequest(r), i)
}

func BindHttpQuery(r *http.Request, i interface{}) error {
	return BindQuery(NewHttpBindableRequest(r), i)
}

func BindHttpHeader(r *http.Request, i interface{}) error {
	return BindHeader(NewHttpBindableRequest(r), i)
}

func BindHttpForm(r *http.Request, i interface{}) error {
	return BindForm(NewHttpBindableRequest(r), i)
}

func BindHttpMultipartForm(r *http.Request, i interface{}) error {
	return BindMultipartForm(NewHttpBindableRequest(r), i)
}
