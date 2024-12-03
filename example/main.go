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
	Test string `json:"test" xml:"test" form:"test" query:"test"`
}

type TestStruct struct {
	Name  string                `json:"name" xml:"name" form:"name" query:"name" path:"name"`
	Age   int                   `json:"age" xml:"age" form:"age" query:"age"`
	Email string                `json:"email" xml:"email" form:"email" query:"email"`
	File  *multipart.FileHeader `json:"file" xml:"file" form:"file"`
	Inner InnerStruct           `json:"inner" xml:"inner" form:"inner"`
}

func handler(b binder.Binder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := &TestStruct{}
		if err := b.Bind(r, data); err != nil {
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
	b := binder.NewBinder()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler(b))
	mux.HandleFunc("/{name}", handler(b))

	log.Fatal(http.ListenAndServe(":8080", mux))
}
