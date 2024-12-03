package binder

import (
	"errors"
	"mime/multipart"
	"net/url"
	"reflect"
	"regexp"
	"strings"
)

// DefaultBinder is the default implementation of the `Binder` interface.
type DefaultBinder struct {
	JSONSerializer       JSONSerializer
	XMLSerializer        XMLSerializer
	PathMatcher          *regexp.Regexp
	ArrayMatcher         *regexp.Regexp
	MapMatcher           *regexp.Regexp
	ArrayNotationMatcher *regexp.Regexp
	MaxBodySize          int64
	MaxArraySize         int
	HeaderTagName        string
	FormTagName          string
	QueryTagName         string
	ParamTagName         string
	BindOrder            []BindFunc
}

func NewBinder() *DefaultBinder {
	r := &DefaultBinder{
		JSONSerializer:       DefaultJSONSerializer{},
		XMLSerializer:        DefaultXMLSerializer{},
		PathMatcher:          PathMatcherRegexp,
		MaxBodySize:          DefaultBodySize,
		MapMatcher:           MapMatcherRegexp,
		ArrayMatcher:         ArrayMatcherRegexp,
		ArrayNotationMatcher: ArrayNotationRegexp,
		MaxArraySize:         MaxArraySize,
		HeaderTagName:        DefaultHeaderTagName,
		FormTagName:          DefaultFormTagName,
		QueryTagName:         DefaultQueryTagName,
		ParamTagName:         DefaultParamTagName,
		BindOrder:            []BindFunc{},
	}

	r.BindOrder = []BindFunc{
		r.BindPathParams,
		r.BindQueryParams,
		r.BindBody,
	}

	return r
}

func (b *DefaultBinder) GetPathParams(r BindableRequest) map[string][]string {
	pattern := r.GetPathPattern()
	if pattern == "" {
		return nil
	}

	names := b.PathMatcher.FindAllStringSubmatch(pattern, -1)
	if len(names) == 0 {
		return nil
	}

	values := map[string][]string{}
	for _, name := range names {
		values[name[1]] = []string{r.GetPathValue(name[1])}
	}
	return values
}

func (b *DefaultBinder) GetQueryParams(r BindableRequest) map[string][]string {
	return r.GetQuery()
}

func (b *DefaultBinder) GetHeaders(r BindableRequest) map[string][]string {
	return r.GetHeaders()
}

// BindPathParams binds path params to bindable object
func (b *DefaultBinder) BindPathParams(r BindableRequest, i interface{}) error {
	values := b.GetPathParams(r)
	if err := b.bindData(i, values, b.ParamTagName, nil); err != nil {
		return err
	}
	return nil
}

// BindQueryParams binds query params to bindable object
func (b *DefaultBinder) BindQueryParams(r BindableRequest, i interface{}) error {
	values := b.GetQueryParams(r)
	if err := b.bindData(i, values, b.QueryTagName, nil); err != nil {
		return err
	}
	return nil
}

// BindBody binds request body contents to bindable object
// NB: then binding forms take note that this implementation uses standard library form parsing
// which parses form data from BOTH URL and BODY if content type is not MIMEMultipartForm
// See non-MIMEMultipartForm: https://golang.org/pkg/net/http/#Request.ParseForm
// See MIMEMultipartForm: https://golang.org/pkg/net/http/#Request.ParseMultipartForm
func (b *DefaultBinder) BindBody(r BindableRequest, i interface{}) (err error) {
	if r.GetContentLength() <= 0 {
		return
	}
	// return

	// mediatype is found like `mime.ParseMediaType()` does it
	base, _, _ := strings.Cut(r.GetHeaders().Get(HeaderContentType), ";")
	mediatype := strings.TrimSpace(base)

	switch mediatype {
	case MIMEApplicationJSON:
		if err = b.JSONSerializer.Deserialize(r, i); err != nil {
			return err
		}
	case MIMEApplicationXML, MIMETextXML:
		if err = b.XMLSerializer.Deserialize(r, i); err != nil {
			return err
		}
	case MIMEApplicationForm:
		var form url.Values
		if form, err = r.GetForm(); err != nil {
			return err
		}

		if err = b.bindData(i, form, b.FormTagName, nil); err != nil {
			return err
		}
	case MIMEMultipartForm:
		var params *multipart.Form
		if params, err = r.GetMultipartForm(b.MaxBodySize); err != nil {
			return err
		}
		if err = b.bindData(i, params.Value, b.FormTagName, params.File); err != nil {
			return err
		}
	default:
		return errors.New("unsupported media type")
	}
	return nil
}

// BindHeaders binds HTTP headers to a bindable object
func (b *DefaultBinder) BindHeaders(r BindableRequest, i interface{}) error {
	if err := b.bindData(i, r.GetHeaders(), b.FormTagName, nil); err != nil {
		return err
	}
	return nil
}

// Bind implements the `Binder#Bind` function.
// Binding is done in following order: 1) path params; 2) query params; 3) request body. Each step COULD override previous
// step binded values. For single source binding use their own methods BindBody, BindQueryParams, BindPathParams.
func (b *DefaultBinder) Bind(r BindableRequest, i interface{}) (err error) {
	for _, bindFunc := range b.BindOrder {
		if err = bindFunc(r, i); err != nil {
			return err
		}
	}

	return nil
}

// bindData will bind data ONLY fields in destination struct that have EXPLICIT tag
func (b *DefaultBinder) bindData(destination interface{}, data map[string][]string, tag string, dataFiles map[string][]*multipart.FileHeader) error {
	if destination == nil || (len(data) == 0 && len(dataFiles) == 0) {
		return nil
	}
	hasFiles := len(dataFiles) > 0
	typ := reflect.TypeOf(destination).Elem()
	val := reflect.ValueOf(destination).Elem()

	// Support binding to limited Map destinations:
	// - map[string][]string,
	// - map[string]string <-- (binds first value from data slice)
	// - map[string]interface{}
	// You are better off binding to struct but there are user who want this map feature. Source of data for these cases are:
	// params,query,header,form as these sources produce string values, most of the time slice of strings, actually.
	if typ.Kind() == reflect.Map && typ.Key().Kind() == reflect.String {
		k := typ.Elem().Kind()
		isElemInterface := k == reflect.Interface
		isElemString := k == reflect.String
		isElemSliceOfStrings := k == reflect.Slice && typ.Elem().Elem().Kind() == reflect.String
		if !(isElemSliceOfStrings || isElemString || isElemInterface) {
			return nil
		}
		if val.IsNil() {
			val.Set(reflect.MakeMap(typ))
		}
		for k, v := range data {
			if isElemString {
				val.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v[0]))
			} else if isElemInterface {
				// To maintain backward compatibility, we always bind to the first string value
				// and not the slice of strings when dealing with map[string]interface{}{}
				val.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v[0]))
			} else {
				val.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
			}
		}
		return nil
	}

	// !struct
	if typ.Kind() != reflect.Struct {
		if tag == b.ParamTagName || tag == b.QueryTagName || tag == b.HeaderTagName {
			// incompatible type, data is probably to be found in the body
			return nil
		}
		return errors.New("binding element must be a struct")
	}

	for i := 0; i < typ.NumField(); i++ { // iterate over all destination fields
		typeField := typ.Field(i)
		structField := val.Field(i)
		if typeField.Anonymous {
			if structField.Kind() == reflect.Ptr {
				structField = structField.Elem()
			}
		}
		if !structField.CanSet() {
			continue
		}
		structFieldKind := structField.Kind()
		inputFieldName := typeField.Tag.Get(tag)
		// valueKind := structField.Type().Kind()

		if typeField.Anonymous && structFieldKind == reflect.Struct && inputFieldName != "" {
			// if anonymous struct with query/param/form tags, report an error
			return errors.New("query/param/form tags are not allowed with anonymous struct field")
		}

		if inputFieldName == "" {
			// If tag is nil, we inspect if the field is a not BindUnmarshaler struct and try to bind data into it (might contain fields with tags).
			// structs that implement BindUnmarshaler are bound only when they have explicit tag
			if _, ok := structField.Addr().Interface().(BindUnmarshaler); !ok && structFieldKind == reflect.Struct {
				if err := b.bindData(structField.Addr().Interface(), data, tag, dataFiles); err != nil {
					return err
				}
			}
			// does not have explicit tag and is not an ordinary struct - so move to next field
			continue
		}

		if hasFiles {
			if ok, err := isFieldMultipartFile(structField.Type()); err != nil {
				return err
			} else if ok {
				if ok := setMultipartFileHeaderTypes(structField, inputFieldName, dataFiles); ok {
					continue
				}
			}
		}

		//if the field is a struct, we need to recursively bind data to it
		if structFieldKind == reflect.Struct {
			// the data now is only the data that is relevant to the current struct
			structData := trimData(inputFieldName, data, b.ArrayNotationMatcher)
			structFiles := trimFileFields(inputFieldName, dataFiles, b.ArrayNotationMatcher)
			if err := b.bindData(structField.Addr().Interface(), structData, tag, structFiles); err != nil {
				return err
			}
			continue
		} else if structFieldKind == reflect.Map {
			// the data now is only the data that is relevant to the current field
			mapData := trimData(inputFieldName, data, b.MapMatcher)
			mapFiles := trimFileFields(inputFieldName, dataFiles, b.MapMatcher)
			if err := b.bindData(structField.Addr().Interface(), mapData, tag, mapFiles); err != nil {
				return err
			}
			// continue
		} else if structFieldKind == reflect.Slice {
			// the data now is only the data that is relevant to the current field

			sliceData := trimData(inputFieldName, data, b.ArrayMatcher)
			sliceFiles := trimFileFields(inputFieldName, dataFiles, b.ArrayMatcher)
			if err := handleArrayValues(structField, structFieldKind, sliceData, sliceFiles, inputFieldName, b.MaxArraySize); err != nil {
				return err
			}
		}

		inputValue, exists := data[inputFieldName]
		if !exists {
			// Go json.Unmarshal supports case-insensitive binding.  However the
			// url params are bound case-sensitive which is inconsistent.  To
			// fix this we must check all of the map values in a
			// case-insensitive search.
			for k, v := range data {

				if strings.EqualFold(k, inputFieldName) {
					inputValue = v
					exists = true
					break
				}
			}
		}

		if !exists {
			continue
		}

		// NOTE: algorithm here is not particularly sophisticated. It probably does not work with absurd types like `**[]*int`
		// but it is smart enough to handle niche cases like `*int`,`*[]string`,`[]*int` .

		// try unmarshalling first, in case we're dealing with an alias to an array type
		if ok, err := unmarshalInputsToField(typeField.Type.Kind(), inputValue, structField); ok {
			if err != nil {
				return err
			}
			continue
		}

		if ok, err := unmarshalInputToField(typeField.Type.Kind(), inputValue[0], structField); ok {
			if err != nil {
				return err
			}
			continue
		}

		// we could be dealing with pointer to slice `*[]string` so dereference it. There are wierd OpenAPI generators
		// that could create struct fields like that.
		if structFieldKind == reflect.Pointer {
			structFieldKind = structField.Elem().Kind()
			structField = structField.Elem()
		}

		if structFieldKind == reflect.Slice {
			sliceOf := structField.Type().Elem().Kind()
			numElems := len(inputValue)
			slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
			for j := 0; j < numElems; j++ {
				if err := setWithProperType(sliceOf, inputValue[j], slice.Index(j)); err != nil {
					return err
				}
			}
			structField.Set(slice)
			continue
		}

		if err := setWithProperType(structFieldKind, inputValue[0], structField); err != nil {
			return err
		}
	}
	return nil
}
