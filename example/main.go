package main

import (
	"encoding/json"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gobigbang/binder"
)

type InnerStruct struct {
	Test string                 `json:"test" xml:"test" form:"test" query:"test"`
	File *multipart.FileHeader  `json:"file" xml:"file" form:"file"`
	Age  int                    `json:"age" xml:"age" form:"age"`
	Map  map[string]interface{} `json:"map" xml:"map" form:"map"`
}

type TestStruct struct {
	Name        string                 `json:"name" xml:"name" form:"name" query:"name" path:"name"`
	HeaderValue string                 `json:"header_value" xml:"header_value" form:"header_value" query:"header_value" header:"X-Header-Value"`
	Age         int                    `json:"age" xml:"age" form:"age" query:"age"`
	FloatNumber *float64               `json:"float_number" xml:"float_number" form:"float_number" query:"float_number"`
	Email       string                 `json:"email" xml:"email" form:"email" query:"email"`
	File        *multipart.FileHeader  `json:"file" xml:"file" form:"file"`
	Inner       InnerStruct            `json:"inner" xml:"inner" form:"inner" query:"inner"`
	Elements    []int                  `json:"elements" xml:"elements" form:"elements[]" query:"elements[]"`
	Map         map[string]interface{} `json:"map" xml:"map" form:"map"`
}

func handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := &TestStruct{}
		if err := binder.BindHttp(r, data); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if data.File != nil {
			file, err := data.File.Open()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// serve the file
			http.ServeContent(w, r, data.File.Filename, time.Now(), file)
			return
		}
		// data := make(map[string]interface{})

		// if err := b.Bind(r, &data); err != nil {
		// 	http.Error(w, err.Error(), http.StatusBadRequest)
		// 	return
		// }

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(data)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler())
	mux.HandleFunc("/{name}", handler())

	log.Fatal(http.ListenAndServe(":8080", mux))
}
