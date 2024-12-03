
[![GoDoc](https://godoc.org/github.com/gobigbang/binder?status.svg)](https://godoc.org/github.com/gobigbang/binder)
[![GitHub release](https://img.shields.io/github/release/gobigbang/binder.svg?v0.0.2)](https://img.shields.io/github/release/gobigbang/binder.svg?v0.0.2)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/gobigbang/binder/master/LICENSE)

# HTTP BINDER

**Golang net/http compatible request binder!**

> This is a modified [Echo](https://github.com/labstack/echo) http [binder](https://github.com/labstack/echo/blob/fe2627778114fc774a1b10920e1cd55fdd97cf00/binder.go) to bind structs and maps from the net/http requests

Parsing request data is a crucial part of a web application, this process is usually called _binding_, and it'is done with information passed by the client in the following parts of an HTTP request:

- URL Path parameter
- URL Query parameter
- Header
- Request body

This package provides different ways to "bind" request data to go types.


# Installation
To install Binder, use go get:

```go
go get github.com/gobigbang/binder
```

# Struct Tag Binding

With struct binding you define a Go struct with tags specifying the data source and corresponding key. In your request handler you simply call `binder.BindHttp(r *http.Request, i interface{})` with a pointer to your struct. The tags tell the binder everything it needs to know to load data from the request.

In this example a struct type `User` tells the binder to bind the query string parameter `id` to its string field `ID`:

```go
type User struct {
  ID string `query:"id"`
}

// bind this route to something like /user?id=abc
func(w http.ResponseWriter, r *http.Request) {
    var user User
    if err := binder.BindHttp(r, &user); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // do something with the user
}

```

### Data Sources

The binder supports the following struct tags to bind the source:

- `query` - query parameter
- `param` - path parameter (also called route)
- `header` - header parameter
- `json` - request body. Uses builtin Go [json](https://golang.org/pkg/encoding/json/) package for unmarshalling.
- `xml` - request body. Uses builtin Go [xml](https://golang.org/pkg/encoding/xml/) package for unmarshalling.
- `form` - form data. Values are taken from query and request body. Uses Go standard library form parsing.

You can modify the tag binding name on the binder instance.

### Data Types

When decoding the request body, the following data types are supported as specified by the `Content-Type` header:

- `application/json`
- `application/xml`
- `application/x-www-form-urlencoded`
- `multipart/form-data`

When binding path parameter, query parameter, header, or form data, tags must be explicitly set on each struct field. However, JSON and XML binding is done on the struct field name if the tag is omitted. This is according to the behavior of [Go's json package](https://pkg.go.dev/encoding/json#Unmarshal).

For form data, the package parses form data from both the request URL and body if content type is not `MIMEMultipartForm`. See documentation for [non-MIMEMultipartForm](https://golang.org/pkg/net/http/#Request.ParseForm)and [MIMEMultipartForm](https://golang.org/pkg/net/http/#Request.ParseMultipartForm)

### Multiple Sources

It is possible to specify multiple sources on the same field. In this case request data is bound in this order (by default):

1. Path parameters
2. Query parameters
3. Request body

```go
type User struct {
  ID string `param:"id" query:"id" form:"id" json:"id" xml:"id"`
}
```

Note that binding at each stage will overwrite data bound in a previous stage. This means if your JSON request contains the query param `name=query` and body `{"name": "body"}` then the result will be `User{Name: "body"}`.

> [!NOTE]
> Please note that BindHeaders is not enabled by default, you must enable it manually or
> call `binder.BindHeader` specifically.

### Security

To keep your application secure, avoid passing bound structs directly to other methods if these structs contain fields that should not be bindable. It is advisable to have a separate struct for binding and map it explicitly to your business struct.

Consider what will happen if your bound struct has an Exported field `IsAdmin bool` and the request body contains `{IsAdmin: true, Name: "hacker"}`.

### Example

In this example we define a `User` struct type with field tags to bind from `json`, `form`, or `query` request data:

```go
type UserDTO struct {
  Name  string `json:"name" form:"name" query:"name"`
  Email string `json:"email" form:"email" query:"email"`
}

type User struct {
  Name    string
  Email   string
  IsAdmin bool
}
```

And a handler at the POST `/users` route binds request data to the struct:

```go
func(w http.ResponseWriter, r *http.Request) {
  u := new(UserDTO)
  if err := binder.BindHttp(r, u); err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }

  // Load into separate struct for security
  user := User{
    Name: u.Name,
    Email: u.Email,
    IsAdmin: false // avoids exposing field that should not be bound
  }

  executeSomeBusinessLogic(user)

  // return something
}
```

#### JSON Data

```sh
curl -X POST http://localhost:1323/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"Joe","email":"joe@labstack"}'
```

#### Form Data

```sh
curl -X POST http://localhost:1323/users \
  -d 'name=Joe' \
  -d 'email=joe@labstack.com'
```

#### Query Parameters

```sh
curl -X GET 'http://localhost:1323/users?name=Joe&email=joe@labstack.com'
```

### Supported Data Types

| Data Type           | Notes                                                                                                          |
| ------------------- | -------------------------------------------------------------------------------------------------------------- |
| `bool`              |                                                                                                                |
| `float32`           |                                                                                                                |
| `float64`           |                                                                                                                |
| `int`               |                                                                                                                |
| `int8`              |                                                                                                                |
| `int16`             |                                                                                                                |
| `int32`             |                                                                                                                |
| `int64`             |                                                                                                                |
| `uint`              |                                                                                                                |
| `uint8/byte`        | Does not support `bytes()`. Use `BindUnmarshaler`/`CustomFunc` to convert value from base64 etc to `[]byte{}`. |
| `uint16`            |                                                                                                                |
| `uint32`            |                                                                                                                |
| `uint64`            |                                                                                                                |
| `string`            |                                                                                                                |
| `time`              |                                                                                                                |
| `duration`          |                                                                                                                |
| `BindUnmarshaler()` | binds to a type implementing BindUnmarshaler interface                                                         |
| `TextUnmarshaler()` | binds to a type implementing encoding.TextUnmarshaler interface                                                |
| `JsonUnmarshaler()` | binds to a type implementing json.Unmarshaler interface                                                        |
| `UnixTime()`        | converts Unix time (integer) to `time.Time`                                                                    |
| `UnixTimeMilli()`   | converts Unix time with millisecond precision (integer) to `time.Time`                                         |
| `UnixTimeNano()`    | converts Unix time with nanosecond precision (integer) to `time.Time`                                          |
| `CustomFunc()`      | callback function for your custom conversion logic                                                             |

Each supported type has the following methods:

- `<Type>("param", &destination)` - if parameter value exists then binds it to given destination of that type i.e `Int64(...)`.
- `Must<Type>("param", &destination)` - parameter value is required to exist, binds it to given destination of that type i.e `MustInt64(...)`.
- `<Type>s("param", &destination)` - (for slices) if parameter values exists then binds it to given destination of that type i.e `Int64s(...)`.
- `Must<Type>s("param", &destination)` - (for slices) parameter value is required to exist, binds it to given destination of that type i.e `MustInt64s(...)`.

For certain slice types `BindWithDelimiter("param", &dest, ",")` supports splitting parameter values before type conversion is done. For example binding an integer slice from the URL `/api/search?id=1,2,3&id=1` will result in `[]int64{1,2,3,1}`.


## License

MIT License