package binder_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gobigbang/binder"
)

type TestStruct struct {
	Name  string `json:"name" xml:"name" form:"name"`
	Age   int    `json:"age" xml:"age" form:"age"`
	Email string `json:"email" xml:"email" form:"email"`
}

func TestBindBody(t *testing.T) {
	binder := binder.NewBinder()

	t.Run("JSON", func(t *testing.T) {
		body := `{"name":"John Doe","age":30,"email":"john@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		var data TestStruct
		err := binder.BindBody(req, &data)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if data.Name != "John Doe" || data.Age != 30 || data.Email != "john@example.com" {
			t.Fatalf("expected data to be bound correctly, got %+v", data)
		}
	})

	t.Run("XML", func(t *testing.T) {
		body := `<TestStruct><name>John Doe</name><age>30</age><email>john@example.com</email></TestStruct>`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/xml")

		var data TestStruct
		err := binder.BindBody(req, &data)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if data.Name != "John Doe" || data.Age != 30 || data.Email != "john@example.com" {
			t.Fatalf("expected data to be bound correctly, got %+v", data)
		}
	})

	t.Run("Form", func(t *testing.T) {
		form := url.Values{}
		form.Add("name", "John Doe")
		form.Add("age", "30")
		form.Add("email", "john@example.com")
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var data TestStruct
		err := binder.BindBody(req, &data)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if data.Name != "John Doe" || data.Age != 30 || data.Email != "john@example.com" {
			t.Fatalf("expected data to be bound correctly, got %+v", data)
		}
	})

	t.Run("MultipartForm", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		writer.WriteField("name", "John Doe")
		writer.WriteField("age", "30")
		writer.WriteField("email", "john@example.com")
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		var data TestStruct
		err := binder.BindBody(req, &data)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if data.Name != "John Doe" || data.Age != 30 || data.Email != "john@example.com" {
			t.Fatalf("expected data to be bound correctly, got %+v", data)
		}
	})
}
