// this code was originally cloned from https://github.com/labstack/echo/blob/fe2627778114fc774a1b10920e1cd55fdd97cf00/binder.go
// and modified to add more advanced features

package binder

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"mime/multipart"
	"net/url"
	"regexp"
)

var ArrayMatcherRegexp = regexp.MustCompile(`\[([0-9]+)\]`)              // matches [0] to use in indexed arrays
var MapMatcherRegexp = regexp.MustCompile(`\[([a-zA-Z0-9\-\_\.]+)\]`)    // matches [key] to use in maps and deep objects
var ArrayNotationRegexp = regexp.MustCompile(`\[([a-zA-Z0-9\-\_\.]+)\]`) // matches [id] to use in deep objects
var PathMatcherRegexp = regexp.MustCompile(`\{([^}]+)\}`)                // matches {id} to use in path parameters
var DefaultDeepObjectSeparator = "."                                     // default separator for deep fields
var DefaultBodySize = int64(32 << 20)                                    // 32 MB
var DefaultHeaderTagName = "header"                                      // default tag name for header
var DefaultFormTagName = "form"                                          // default tag name for form
var DefaultQueryTagName = "query"                                        // default tag name for query
var DefaultParamTagName = "param"                                        // default tag name for param
var MaxArraySize = 1000                                                  // max size of array

// JSONSerializer is the interface that encodes and decodes JSON to and from interfaces.
type JSONSerializer interface {
	// Serialize(c context.Context, i interface{}, indent string) error
	Deserialize(r BindableRequest, i interface{}) error
}

// BindableRequest is the interface that wraps the basic methods required for
// a request to be bindable.
//
// This enables non-HTTP request types to be bindable.
type BindableRequest interface {
	GetBody() io.Reader
	GetPathPattern() string
	GetPathValue(string) string
	GetQuery() url.Values
	GetHeaders() url.Values
	GetContentLength() int64
	GetContentType() string
	GetForm() (url.Values, error)
	GetMultipartForm(maxBodySize int64) (*multipart.Form, error)
}

type BindFunc func(r BindableRequest, i interface{}) error

type DefaultJSONSerializer struct{}

func (DefaultJSONSerializer) Deserialize(r BindableRequest, i interface{}) error {
	return json.NewDecoder(r.GetBody()).Decode(i)
}

type XMLSerializer interface {
	Deserialize(r BindableRequest, i interface{}) error
}

type DefaultXMLSerializer struct{}

func (DefaultXMLSerializer) Deserialize(r BindableRequest, i interface{}) error {
	return xml.NewDecoder(r.GetBody()).Decode(i)
}

type Binder interface {
	Bind(r BindableRequest, i interface{}) error
	BindBody(r BindableRequest, i interface{}) error
	BindPathParams(r BindableRequest, i interface{}) error
	BindQueryParams(r BindableRequest, i interface{}) error
	BindHeaders(r BindableRequest, i interface{}) error
}

// BindUnmarshaler is the interface used to wrap the UnmarshalParam method.
// Types that don't implement this, but do implement encoding.TextUnmarshaler
// will use that interface instead.
type BindUnmarshaler interface {
	// UnmarshalParam decodes and assigns a value from an form or query param.
	UnmarshalParam(param string) error
}

// bindMultipleUnmarshaler is used by binder to unmarshal multiple values from request at once to
// type implementing this interface. For example request could have multiple query fields `?a=1&a=2&b=test` in that case
// for `a` following slice `["1", "2"] will be passed to unmarshaller.
type bindMultipleUnmarshaler interface {
	UnmarshalParams(params []string) error
}

var DefaultBinderInstance Binder

// Returns the default binder instance.
func GetBinder() Binder {
	if DefaultBinderInstance == nil {
		DefaultBinderInstance = NewBinder()
	}
	return DefaultBinderInstance
}

func Bind(r BindableRequest, i interface{}) error {
	return GetBinder().Bind(r, i)
}

func BindBody(r BindableRequest, i interface{}) error {
	return GetBinder().BindBody(r, i)
}

func BindPathParams(r BindableRequest, i interface{}) error {
	return GetBinder().BindPathParams(r, i)
}

func BindQueryParams(r BindableRequest, i interface{}) error {
	return GetBinder().BindQueryParams(r, i)
}

func BindHeaders(r BindableRequest, i interface{}) error {
	return GetBinder().BindHeaders(r, i)
}
