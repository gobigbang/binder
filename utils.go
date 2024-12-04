package binder

import (
	"encoding"
	"errors"
	"fmt"
	"mime/multipart"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// getPrefixedFieldNames returns a map of field names that are prefixed with the given prefix.
func getPrefixedFieldNames(prefix string, keys []string, matcher *regexp.Regexp, deepSeparator string) map[string]string {
	result := map[string]string{}
	for _, k := range keys {
		if strings.HasPrefix(k, prefix) {
			if strings.HasPrefix(k, prefix+deepSeparator) {
				result[k] = strings.TrimPrefix(k, prefix+deepSeparator) // dot notation
			} else if matches := matcher.FindAllStringSubmatch(k, -1); len(matches) > 0 {
				if len(matches) == 0 {
					continue
				}
				finalValue := []string{}
				// convert all the matches to dot notation (it should be faster than using check for each match)
				for _, match := range matches {
					val := match[1]
					if val == "" {
						break
					}
					finalValue = append(finalValue, val)
				}
				result[k] = strings.Join(finalValue, deepSeparator)
			}
		}
	}
	return result
}

// trimData trims the data map to only include keys that start with the given prefix.
func trimData(prefix string, data map[string][]string, matcher *regexp.Regexp, deepSeparator string) map[string][]string {
	result := map[string][]string{}
	keys := []string{}
	for key := range data {
		keys = append(keys, key)
	}
	fieldNames := getPrefixedFieldNames(prefix, keys, matcher, deepSeparator)
	for k, v := range fieldNames {
		result[v] = data[k]
	}
	return result
}

// trimFileFields trims the files map to only include keys that start with the given prefix.
func trimFileFields(prefix string, files map[string][]*multipart.FileHeader, matcher *regexp.Regexp, deepSeparator string) map[string][]*multipart.FileHeader {
	result := map[string][]*multipart.FileHeader{}
	keys := []string{}
	for key := range files {
		keys = append(keys, key)
	}
	fieldNames := getPrefixedFieldNames(prefix, keys, matcher, deepSeparator)
	for k, v := range fieldNames {
		result[v] = files[k]
	}
	for k, v := range fieldNames {
		result[v] = files[k]
	}
	return result
}

func setWithProperType(valueKind reflect.Kind, val string, structField reflect.Value) error {
	// But also call it here, in case we're dealing with an array of BindUnmarshalers
	if ok, err := unmarshalInputToField(valueKind, val, structField); ok {
		return err
	}

	switch valueKind {
	case reflect.Ptr:
		return setWithProperType(structField.Elem().Kind(), val, structField.Elem())
	case reflect.Int:
		return setIntField(val, 0, structField)
	case reflect.Int8:
		return setIntField(val, 8, structField)
	case reflect.Int16:
		return setIntField(val, 16, structField)
	case reflect.Int32:
		return setIntField(val, 32, structField)
	case reflect.Int64:
		return setIntField(val, 64, structField)
	case reflect.Uint:
		return setUintField(val, 0, structField)
	case reflect.Uint8:
		return setUintField(val, 8, structField)
	case reflect.Uint16:
		return setUintField(val, 16, structField)
	case reflect.Uint32:
		return setUintField(val, 32, structField)
	case reflect.Uint64:
		return setUintField(val, 64, structField)
	case reflect.Bool:
		return setBoolField(val, structField)
	case reflect.Float32:
		return setFloatField(val, 32, structField)
	case reflect.Float64:
		return setFloatField(val, 64, structField)
	case reflect.String:
		structField.SetString(val)
	default:
		return errors.New("unknown type")
	}
	return nil
}

func unmarshalInputsToField(valueKind reflect.Kind, values []string, field reflect.Value) (bool, error) {
	if valueKind == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	fieldIValue := field.Addr().Interface()
	unmarshaler, ok := fieldIValue.(bindMultipleUnmarshaler)
	if !ok {
		return false, nil
	}
	return true, unmarshaler.UnmarshalParams(values)
}

func unmarshalInputToField(valueKind reflect.Kind, val string, field reflect.Value) (bool, error) {
	if valueKind == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	fieldIValue := field.Addr().Interface()
	switch unmarshaler := fieldIValue.(type) {
	case BindUnmarshaler:
		return true, unmarshaler.UnmarshalParam(val)
	case encoding.TextUnmarshaler:
		return true, unmarshaler.UnmarshalText([]byte(val))
	}

	return false, nil
}

func setIntField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	intVal, err := strconv.ParseInt(value, 10, bitSize)
	if err == nil {
		field.SetInt(intVal)
	}
	return err
}

func setUintField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	uintVal, err := strconv.ParseUint(value, 10, bitSize)
	if err == nil {
		field.SetUint(uintVal)
	}
	return err
}

func setBoolField(value string, field reflect.Value) error {
	if value == "" {
		value = "false"
	}
	boolVal, err := strconv.ParseBool(value)
	if err == nil {
		field.SetBool(boolVal)
	}
	return err
}

func setFloatField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0.0"
	}
	floatVal, err := strconv.ParseFloat(value, bitSize)
	if err == nil {
		field.SetFloat(floatVal)
	}
	return err
}

var (
	// NOT supported by bind as you can NOT check easily empty struct being actual file or not
	multipartFileHeaderType = reflect.TypeOf(multipart.FileHeader{})
	// supported by bind as you can check by nil value if file existed or not
	multipartFileHeaderPointerType      = reflect.TypeOf(&multipart.FileHeader{})
	multipartFileHeaderSliceType        = reflect.TypeOf([]multipart.FileHeader(nil))
	multipartFileHeaderPointerSliceType = reflect.TypeOf([]*multipart.FileHeader(nil))
)

func isFieldMultipartFile(field reflect.Type) (bool, error) {
	switch field {
	case multipartFileHeaderPointerType,
		multipartFileHeaderSliceType,
		multipartFileHeaderPointerSliceType:
		return true, nil
	case multipartFileHeaderType:
		return true, errors.New("binding to multipart.FileHeader struct is not supported, use pointer to struct")
	default:
		return false, nil
	}
}

func setMultipartFileHeaderTypes(structField reflect.Value, inputFieldName string, files map[string][]*multipart.FileHeader) bool {
	fileHeaders := files[inputFieldName]
	if len(fileHeaders) == 0 {
		return false
	}

	result := true
	switch structField.Type() {
	case multipartFileHeaderPointerSliceType:
		structField.Set(reflect.ValueOf(fileHeaders))
	case multipartFileHeaderSliceType:
		headers := make([]multipart.FileHeader, len(fileHeaders))
		for i, fileHeader := range fileHeaders {
			headers[i] = *fileHeader
		}
		structField.Set(reflect.ValueOf(headers))
	case multipartFileHeaderPointerType:
		structField.Set(reflect.ValueOf(fileHeaders[0]))
	default:
		result = false
	}

	return result
}

func handleArrayValues(structValue reflect.Value, structFieldKind reflect.Kind, values map[string][]string, _ map[string][]*multipart.FileHeader, inputFieldName string, maxArraySize int) error {
	if structFieldKind == reflect.Slice {
		for k, v := range values {
			intIndex, err := strconv.Atoi(k)
			if err != nil {
				return fmt.Errorf("invalid array index %s", k)
			}

			if intIndex > maxArraySize {
				return fmt.Errorf("%s array size exceeds the maximum allowed size of %d", inputFieldName, maxArraySize)
			}

			// check if the slice has already been created
			slice := structValue
			if slice.Len() == 0 {
				// create a slice with enough capacity
				slice = reflect.MakeSlice(structValue.Type(), intIndex+1, intIndex+1)
			} else if slice.Len() <= intIndex {
				// create a new slice with enough capacity
				newSlice := reflect.MakeSlice(structValue.Type(), intIndex+1, intIndex+1)
				reflect.Copy(newSlice, slice)
				slice = newSlice
			}
			if err := setWithProperType(structValue.Type().Elem().Kind(), v[0], slice.Index(intIndex)); err != nil {
				return err
			}

			structValue.Set(slice)
		}
	}
	return nil
}
