package api

import (
	"io"
	"net/url"
	"strconv"

	"github.com/jcdotter/go/buffer"
	"github.com/jcdotter/go/data"
	"github.com/jcdotter/go/encoder"
)

// ----------------------------------------------------------------------------
// API TYPES

// Protocol is the protocol used to
// communicate with the API
type Protocol byte

const (
	REST Protocol = iota
	GRPC
	WEBSOCKET
	SOAP
)

// ----------------------------------------------------------------------------
// METHOD TYPES

// MethodType is the HTTP method
type MethodType byte

const (
	GET MethodType = iota
	POST
	PUT
	DELETE
	PATCH
)

var (
	methodIndex  = []int{0, 4, 9, 13, 20}
	methodString = "GET POST PUT DELETE PATCH "
	methodType   = map[string]MethodType{
		"GET":    GET,
		"POST":   POST,
		"PUT":    PUT,
		"DELETE": DELETE,
		"PATCH":  PATCH,
	}
)

func Methodtype(s string) MethodType {
	return methodType[s]
}

func (t MethodType) String() string {
	return methodString[methodIndex[t]:methodIndex[t+1]]
}

// ----------------------------------------------------------------------------
// CONTENT TYPES

// ContentType is the type of the
// content in the request or response
type ContentType byte

const (
	JSON ContentType = iota
	XML
	FORM
	TEXT
	CSV
	HTML
	XLS
	DOC
)

var (
	contentString = map[ContentType]string{
		JSON: "application/json",
		XML:  "application/xml",
		FORM: "application/x-www-form-urlencoded",
		TEXT: "text/plain",
		CSV:  "text/csv",
		HTML: "text/html",
		XLS:  "application/vnd.ms-excel",
		DOC:  "application/msword",
	}
	contentType = map[string]ContentType{
		"application/json":                  JSON,
		"application/xml":                   XML,
		"application/x-www-form-urlencoded": FORM,
		"text/plain":                        TEXT,
		"text/csv":                          CSV,
		"text/html":                         HTML,
		"application/vnd.ms-excel":          XLS,
		"application/msword":                DOC,
	}
)

func Contenttype(s string) ContentType {
	return contentType[s]
}

func (t ContentType) String() string {
	return contentString[t]
}

// ----------------------------------------------------------------------------
// DATA TYPES

// DataType is the type of a data element
// in the request or response
type DataType byte

const (
	NONE DataType = iota
	BOOL
	INT
	FLOAT
	STRING
	LIST
	OBJECT
	ANY
)

var (
	dataTypeString = "noneboolintfloatstringlistobjectany"
	dataTypeIndex  = []int{0, 4, 8, 11, 16, 22, 26, 32, 35}
	dataType       = map[string]DataType{
		"none":   NONE,
		"bool":   BOOL,
		"int":    INT,
		"float":  FLOAT,
		"string": STRING,
		"list":   LIST,
		"object": OBJECT,
		"any":    ANY,
	}
)

type Object map[string]any
type List []any

func DataTypeOf(s string) DataType {
	return dataType[s]
}

func (d DataType) String() string {
	return dataTypeString[dataTypeIndex[d]:dataTypeIndex[d+1]]
}

// ----------------------------------------------------------------------------
// API
// api.resource.method.request

type Api struct {
	Protocol  Protocol
	Auth      *Api
	Resources *data.Data
}

func New() *Api {
	return &Api{
		Protocol:  REST,
		Resources: data.Make[*Resource](4),
	}
}

func FromYaml(yaml []byte) *Api {
	return FromMap(encoder.Yaml.Decode(yaml).Map())
}

func FromMap(m map[string]any) (api *Api) {
	if u, ok := m["url"]; ok {
		if url, e := url.Parse(u.(string)); e == nil {
			api = New()
			if p, ok := m["auth"]; ok {
				api.Auth = FromMap(p.(map[string]any))
			}
			if r, ok := m["resources"]; ok {
				for k, v := range r.(map[string]any) {
					api.ResourceMap(k, v.(map[string]any), url)
				}
			}
		}
	}
	return
}

func (a *Api) ResourceMap(k string, m map[string]any, u *url.URL) {
	if uri, ok := m["uri"]; ok {
		u.Path = uri.(string)
		r := NewResource(a, k, u)
		a.Resources.Add(r)
		if ms, ok := m["methods"]; ok {
			for k, v := range ms.(map[string]any) {
				r.MethodMap(k, v.(map[string]any))
			}
		}
	}
}

func (a *Api) Resource(key string) *Resource {
	var el any
	if el = a.Resources.Get(key); el == nil {
		return nil
	}
	return el.(*Resource)
}

// ----------------------------------------------------------------------------
// API RESOURCE

type Resource struct {
	Api     *Api
	Name    string
	Url     *url.URL
	Methods *data.Data
}

func NewResource(api *Api, name string, url *url.URL) *Resource {
	return &Resource{
		Api:     api,
		Name:    name,
		Url:     url,
		Methods: data.Make[*Method](4),
	}
}

func (r *Resource) MethodMap(k string, m map[string]any) {
	me := NewMethod(r, k)
	if r, ok := m["request"]; ok {
		me.Request = RequestMap(r.(map[string]any))
	}
	if r, ok := m["response"]; ok {
		me.Response = ResponseMap(r.(map[string]any))
	}
	r.Methods.Add(me)
}

func (r *Resource) Key() string {
	return r.Name
}

func (r *Resource) Method(key string) *Method {
	var el any
	if el = r.Methods.Get(key); el == nil {
		return nil
	}
	return el.(*Method)
}

func (r *Resource) Get()    {}
func (r *Resource) Post()   {}
func (r *Resource) Put()    {}
func (r *Resource) Delete() {}

// ----------------------------------------------------------------------------
// API METHOD

type Method struct {
	Resource *Resource
	Name     string
	Request  *Request
	Response *Response
}

func NewMethod(resource *Resource, name string) *Method {
	return &Method{
		Name:     name,
		Request:  &Request{},
		Response: &Response{},
	}
}

func (m *Method) Key() string {
	return m.Name
}

func (m *Method) Call() {
	// use http client to build and make request
	/* c := &http.Client{}
	r, _ := http.NewRequest(m.Name, m.Resource.Url.String(), nil) */

}

type Request struct {
	Params Params
	Header Params
	Body   Params
	// TODO: add webhooks
}

func NewRequest() *Request {
	return &Request{
		Params: Params{data.Make[*Param](4)},
		Header: Params{data.Make[*Param](4)},
		Body:   Params{data.Make[*Param](4)},
	}
}

func RequestMap(m map[string]any) *Request {
	r := &Request{}
	for k, v := range m {
		switch k {
		case "params":
			_, r.Params = ParamMap(v.(map[string]any))
		case "header":
			_, r.Header = ParamMap(v.(map[string]any))
		case "body":
			switch v := v.(type) {
			case map[string]any:
				_, r.Body = ParamMap(v)
			case []any:
				_, r.Body = ParamList(v)
			}
		}
	}
	return r
}

func (r *Request) Reader() io.Reader {
	//b :=
	return nil
}

type Response struct {
	Header Params
	Body   Params
}

func NewResponse() *Response {
	return &Response{
		Header: Params{data.Make[*Param](4)},
		Body:   Params{data.Make[*Param](4)},
	}
}

func ResponseMap(m map[string]any) *Response {
	r := &Response{}
	for k, v := range m {
		switch k {
		case "header":
			_, r.Header = ParamMap(v.(map[string]any))
		case "body":
			switch v := v.(type) {
			case map[string]any:
				_, r.Body = ParamMap(v)
			case []any:
				_, r.Body = ParamList(v)
			}
		}
	}
	return r
}

// ----------------------------------------------------------------------------
// PARAM

type Params struct {
	*data.Data
}

func (p Params) Get(key string) *Param {
	var el any
	if el = p.Data.Get(key); el == nil {
		return nil
	}
	return el.(*Param)
}

func (p Params) Index(i int) *Param {
	var el any
	if el = p.Data.Index(i); el == nil {
		return nil
	}
	return el.(*Param)
}

func (p Params) IsNil() bool {
	return p.Data == nil
}

func (p *Params) Set(k string, v any) *Params {
	if p.Data == nil {
		p.Data = data.Make[*Param](4)
	}
	p.Add(ParamElem(k, v))
	return p
}

// Param is an element of an object or list
// which may be found in the url, header, or
// body of a request or response
type Param struct {
	// if the param belongs to an object
	// the key will be the field name,
	// otherwise the param belongs to a list
	// and the key will be the index
	key string
	// the datatype of the param
	typ DataType
	// if the param is an object or a list with
	// a single datatype and variable length,
	// the elm will be the datatype of the
	// elements in the object or list
	elm DataType
	// if the param is an object or a list
	// the els will be the data elements
	// in the object or list
	els Params
	// if the param is a bool, int, float or string
	// the val will be the value of the param
	val any
}

func ParamMap(m map[string]any) (e DataType, d Params) {
	d = Params{data.Make[*Param](len(m))}
	i := 0
	for k, v := range m {
		p := ParamElem(k, v)
		if i == 0 {
			e = p.typ
		} else if p.typ != e && e != ANY {
			e = ANY
		}
		d.Add(p)
		i++
	}
	return
}

func ParamList(l []any) (e DataType, d Params) {
	d = Params{data.Make[*Param](len(l))}
	for i, v := range l {
		p := ParamElem(strconv.Itoa(i), v)
		if i == 0 {
			e = p.typ
		} else if p.typ != e && e != ANY {
			e = ANY
		}
		d.Add(p)
	}
	return
}

func ParamElem(k string, v any) (p *Param) {
	p = &Param{key: k}
	switch v := v.(type) {
	case bool:
		p.typ = BOOL
		p.val = v
	case int:
		p.typ = INT
		p.val = v
	case float64:
		p.typ = FLOAT
		p.val = v
	case string:
		if p.typ = DataTypeOf(v); p.typ == NONE {
			p.typ = STRING
			p.val = v
		}
	case map[string]any:
		p.typ = OBJECT
		p.elm, p.els = ParamMap(v)
	case []any:
		p.typ = LIST
		p.elm, p.els = ParamList(v)
	}
	return
}

func (p *Param) Key() string {
	return p.key
}

func (p *Param) Type() DataType {
	return p.typ
}

func (p *Param) ElemType() DataType {
	return p.elm
}

func (p *Param) Elem(key string) *Param {
	return p.els.Get(key)
}

func (p *Param) Index(i int) *Param {
	return p.els.Index(i)
}

func (p *Param) Elems() Params {
	return p.els
}

func (p *Param) Len() int {
	return p.els.Len()
}

func (p *Param) Val() any {
	return p.val
}

func (p *Param) Set(a any) bool {
	switch a := a.(type) {
	case bool:
		if p.typ == BOOL {
			p.val = a
			return true
		}
	case int:
		if p.typ == INT {
			p.val = a
			return true
		}
	case float64:
		if p.typ == FLOAT {
			p.val = a
			return true
		}
	case string:
		if p.typ == STRING {
			p.val = a
			return true
		}
	case map[string]any:
		if p.typ == OBJECT {
			if e, d := ParamMap(a); e == p.elm {
				p.els = d
				return true
			}
		}
	case []any:
		if p.typ == LIST {
			if e, d := ParamList(a); e == p.elm {
				p.els = d
				return true
			}
		}
	}
	return false
}

func (p *Params) Json(list ...bool) []byte {
	if p == nil || p.Len() == 0 {
		return nil
	}
	if p.Len() == 1 && p.Index(0).Type() == LIST {
		return p.Index(0).Json()
	}
	b := buffer.New()
	if len(list) > 0 && list[0] {
		for i := 0; i < p.Len(); i++ {
			if v := p.Index(i).Json(); v != nil {
				if i > 0 {
					b.WriteByte(',')
				}
				b.Write(v)
			}
		}
		if b.Len() == 0 {
			return nil
		}
		b.PrependByte('[')
		b.WriteByte(']')
		return b.Bytes()
	}
	b.WriteByte('{')
	for i := 0; i < p.Len(); i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		v := p.Index(i)
		b.WriteString(strconv.Quote(v.Key()))
		b.WriteByte(',')
		b.Write(v.Json())
	}
	b.WriteByte('}')
	return b.Bytes()

}

func (p *Param) Json() []byte {
	switch p.typ {
	case OBJECT:
		return p.els.Json()
	case LIST:
		return p.els.Json(true)
	default:
		if p.val != nil {
			switch p.typ {
			case BOOL:
				return []byte(strconv.FormatBool(p.val.(bool)))
			case INT:
				return []byte(strconv.FormatInt(p.val.(int64), 10))
			case FLOAT:
				return []byte(strconv.FormatFloat(p.val.(float64), 'f', -1, 64))
			case STRING:
				return []byte(strconv.Quote(p.val.(string)))
			}
		}
	}
	return nil
}
