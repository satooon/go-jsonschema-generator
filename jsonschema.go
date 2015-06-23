/*
Basic json-schema generator based on Go types, for easy interchange of Go
structures between diferent languages.
*/
package jsonschema

import (
	"encoding/json"
	"reflect"
	"strings"
)

const (
	DEFAULT_SCHEMA      = "http://json-schema.org/schema#"
	DEFAULT_SCHEMA_TYPE = "object"

	TAG                         = "jschema"
	TAG_BASE                    = "schema"
	TAG_ID                      = "id"
	TAG_DESCRIPTION             = "description"
	TAG_LINKS                   = "links"
	TAG_LINKS_TITLE             = "links-title"
	TAG_LINKS_DESCRIPTION       = "links-description"
	TAG_LINKS_METHOD            = "links-method"
	TAG_LINKS_HREF              = "links-href"
	TAG_LINKS_REL               = "links-rel"
	TAG_LINKS_SCHEMA_PROPERTIES = "links-schema-properties"
	TAG_LINKS_SCHEMA_TYPE       = "links-schema-type"
)

type Document struct {
	Schema      string `json:"$schema,omitempty"`
	Id          string `json:"id,omitempty"`
	Description string `json:"description,omitempty"`
	Links       *links `json:"links,omitempty"`
	property
}

type Base struct {
	Id          string    `json:"-" jschema:"id"`          // API URL
	Description string    `json:"-" jschema:"description"` // Description
	Links       BaseLinks `json:"-" jschema:"links"`
}

type BaseLink struct {
	Title       string `json:"-" jschema:"links-title"`       // API Title
	Description string `json:"-" jschema:"links-description"` // API Description
	Method      string `json:"-" jschema:"links-method"`      // GET or POST
	Href        string `json:"-" jschema:"links-href"`        // Full URL
	Rel         string `json:"-" jschema:"links-rel"`         // self ...
	// Schema      BaseLinksSchema `json:"-" jschema:"links-schema"`
}
type BaseLinks []*BaseLink

// type BaseLinksSchema struct {
// 	property
// }

// Reads the variable structure into the JSON-Schema Document
func (d *Document) Read(variable interface{}) {
	d.setDefaultSchema()

	value := reflect.ValueOf(variable)
	d.read(value.Type(), tagOptions(""))
	d.readId(value)
	d.readDescription(value)
	d.Links.readLinks(value)
	// d.Links.Schema.readLinksSchema(value)
}

func (d *Document) setDefaultSchema() {
	if d.Schema == "" {
		d.Schema = DEFAULT_SCHEMA
	}
	if d.Links == nil {
		d.Links = &links{}
	}
	// if d.Links.Schema == nil {
	// 	d.Links.Schema = &Schema{Type: DEFAULT_SCHEMA_TYPE}
	// }
}

// Marshal returns the JSON encoding of the Document
func (d *Document) Marshal() ([]byte, error) {
	return json.MarshalIndent(d, "", "    ")
}

// String return the JSON encoding of the Document as a string
func (d *Document) String() string {
	json, _ := d.Marshal()
	return string(json)
}

type property struct {
	Type                 string               `json:"type,omitempty"`
	Items                *item                `json:"items,omitempty"`
	Properties           map[string]*property `json:"properties,omitempty"`
	Required             []string             `json:"required,omitempty"`
	AdditionalProperties bool                 `json:"additionalProperties,omitempty"`
}

type link struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Method      string `json:"method,omitempty"`
	Href        string `json:"href,omitempty"`
	Rel         string `json:"rel,omitempty"`
	// Schema      *Schema `json:"schema,omitempty"`
}
type links []*link

// type Schema struct {
// 	Properties map[string]*Schema `json:"properties,omitempty"`
// 	Type       string             `json:"type,omitempty"`
// }

type item struct {
	Type       string               `json:"type,omitempty"`
	Properties map[string]*property `json:"properties,omitempty"`
}

func (p *property) read(t reflect.Type, opts tagOptions) {
	kind := t.Kind()

	if jsType := getTypeFromMapping(kind); jsType != "" {
		p.Type = jsType
	}

	switch kind {
	case reflect.Slice:
		p.readFromSlice(t)
	case reflect.Map:
		p.readFromMap(t)
	case reflect.Struct:
		p.readFromStruct(t)
	case reflect.Ptr:
		p.read(t.Elem(), opts)
	}
}

func (p *property) readFromSlice(t reflect.Type) {
	k := t.Elem().Kind()
	if k == reflect.Uint8 {
		p.Type = "string"
	} else {
		switch t.Elem().Kind() {
		case reflect.Struct:
			p.Items = &item{Type: "object"}
			p.Items.Properties = make(map[string]*property, 0)
			count := t.Elem().NumField()
			for i := 0; i < count; i++ {
				field := t.Elem().Field(i)

				tag := field.Tag.Get(TAG)
				name, opts := parseTag(tag)
				if name != "" {
					continue
				}

				tag = field.Tag.Get("json")
				name, opts = parseTag(tag)
				if name == "" {
					name = field.Name
				}

				p.Items.Properties[name] = &property{}
				p.Items.Properties[name].read(field.Type, opts)
			}
			return
		default:
			if jsType := getTypeFromMapping(k); jsType != "" {
				p.Items = &item{Type: jsType}
			}
		}
	}
}

func (p *property) readFromMap(t reflect.Type) {
	k := t.Elem().Kind()

	if jsType := getTypeFromMapping(k); jsType != "" {
		p.Properties = make(map[string]*property, 0)
		p.Properties[".*"] = &property{Type: jsType}
	} else {
		p.AdditionalProperties = true
	}
}

func (p *property) readFromStruct(t reflect.Type) {
	p.Type = "object"
	p.Properties = make(map[string]*property, 0)
	p.AdditionalProperties = false

	count := t.NumField()
	for i := 0; i < count; i++ {
		field := t.Field(i)

		tag := field.Tag.Get(TAG)
		name, opts := parseTag(tag)
		if name != "" {
			continue
		}

		tag = field.Tag.Get("json")
		name, opts = parseTag(tag)
		if name == "-" {
			continue
		}
		if name == "" {
			name = field.Name
		}

		p.Properties[name] = &property{}
		p.Properties[name].read(field.Type, opts)

		if !opts.Contains("omitempty") {
			if name != "-" {
				p.Required = append(p.Required, name)
			}
		}
	}
}

func (d *Document) readId(v reflect.Value) {
	kind := v.Kind()
	if kind != reflect.Struct {
		return
	}

	count := v.Type().NumField()
	for i := 0; i < count; i++ {
		field := v.Type().Field(i)

		tag := field.Tag.Get(TAG)
		name, opts := parseTag(tag)
		if name == "" {
			continue
		}
		if opts.Contains("omitempty") {
			continue
		}
		if !opts.Contains(TAG_ID) {
			continue
		}
		d.Id = v.String()
	}
}

func (d *Document) readDescription(v reflect.Value) {
	kind := v.Kind()
	if kind != reflect.Struct {
		return
	}

	count := v.Type().NumField()
	for i := 0; i < count; i++ {
		field := v.Type().Field(i)

		tag := field.Tag.Get(TAG)
		name, opts := parseTag(tag)
		if name == "" {
			continue
		}
		if opts.Contains("omitempty") {
			continue
		}
		if !opts.Contains(TAG_DESCRIPTION) {
			continue
		}
		d.Description = v.String()
	}
}

func (l *links) readLinks(v reflect.Value) {
	if !v.IsValid() {
		return
	}

	kind := v.Type().Kind()
	switch kind {
	case reflect.Struct:
		l.readLinksFromStruct(v)
		break
	case reflect.Ptr:
		l.readLinks(v.Elem())
		break
	case reflect.Slice:
		l.readLinksFromSlice(v)
		break
	}
}

func (l *links) readLinksFromStruct(v reflect.Value) {
	if !v.IsValid() {
		return
	}

	count := v.NumField()
	for i := 0; i < count; i++ {
		if v.Field(i).Kind() == reflect.Ptr {
			l.readLinks(v.Field(i))
			continue
		}

		tag, _ := parseTag(v.Type().Field(i).Tag.Get(TAG))
		switch tag {
		case TAG_BASE:
			l.readLinks(v.Field(i))
			continue
		case TAG_LINKS:
			l.readLinks(v.Field(i))
			continue
		}
	}
}

func (l *links) readLinksFromSlice(v reflect.Value) {
	link := &link{}
	for i := 0; i < v.Len(); i++ {
		field := v.Index(i).Elem()
		for j := 0; j < field.NumField(); j++ {
			tag, _ := parseTag(field.Type().Field(j).Tag.Get(TAG))
			switch tag {
			case TAG_LINKS_TITLE:
				link.Title = field.Field(j).String()
				break
			case TAG_LINKS_DESCRIPTION:
				link.Description = field.Field(j).String()
				break
			case TAG_LINKS_METHOD:
				link.Method = field.Field(j).String()
				break
			case TAG_LINKS_HREF:
				link.Href = field.Field(j).String()
				break
			case TAG_LINKS_REL:
				link.Rel = field.Field(j).String()
				break
			}
		}
	}
	json, _ := json.Marshal(link)
	if string(json) != "{}" {
		*l = append(*l, link)
	}
}

// func (s *Schema) readLinksSchema(v reflect.Value) {
// 	if !v.IsValid() {
// 		return
// 	}

// 	kind := v.Type().Kind()
// 	switch kind {
// 	case reflect.Struct:
// 		s.readLinksSchemaFromStruct(v)
// 		break
// 	case reflect.Ptr:
// 		s.readLinksSchema(v.Elem())
// 		break
// 	}
// }

// func (s *Schema) readLinksSchemaFromStruct(v reflect.Value) {
// 	if !v.IsValid() {
// 		return
// 	}

// 	count := v.NumField()
// 	for i := 0; i < count; i++ {
// 		switch v.Field(i).Kind() {
// 		case reflect.Ptr:
// 			s.readLinksSchema(v.Field(i))
// 			continue
// 		case reflect.Struct:
// 			break
// 		}
// 		_, opts := parseTag(v.Type().Field(i).Tag.Get(TAG))
// 		if !opts.Contains(LINKS_SCHEMA_TAB_PROPERTIES) {
// 			continue
// 		}

// 		s.Properties = make(map[string]*Schema, 0)
// 		_count := v.Type().Field(i).Type.NumField()
// 		for j := 0; j < _count; j++ {
// 			_field := v.Type().Field(i).Type.Field(j)

// 			tag := _field.Tag.Get("json")
// 			name, _ := parseTag(tag)
// 			if name == "" {
// 				name = _field.Name
// 			}

// 			s.Properties[name] = &Schema{}
// 			s.Properties[name].readLinksSchemaFromStruct(reflect.ValueOf(v.Type().Field(j)))
// 		}
// 	}
// }

var mapping = map[reflect.Kind]string{
	reflect.Bool:    "bool",
	reflect.Int:     "integer",
	reflect.Int8:    "integer",
	reflect.Int16:   "integer",
	reflect.Int32:   "integer",
	reflect.Int64:   "integer",
	reflect.Uint:    "integer",
	reflect.Uint8:   "integer",
	reflect.Uint16:  "integer",
	reflect.Uint32:  "integer",
	reflect.Uint64:  "integer",
	reflect.Float32: "number",
	reflect.Float64: "number",
	reflect.String:  "string",
	reflect.Slice:   "array",
	reflect.Struct:  "object",
	reflect.Map:     "object",
}

func getTypeFromMapping(k reflect.Kind) string {
	if t, ok := mapping[k]; ok {
		return t
	}

	return ""
}

type tagOptions string

func parseTag(tag string) (string, tagOptions) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tagOptions(tag[idx+1:])
	}
	return tag, tagOptions("")
}

func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}

	s := string(o)
	for s != "" {
		var next string
		i := strings.Index(s, ",")
		if i >= 0 {
			s, next = s[:i], s[i+1:]
		}
		if s == optionName {
			return true
		}
		s = next
	}
	return false
}
