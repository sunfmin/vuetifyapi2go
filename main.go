package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/iancoleman/strcase"
	. "github.com/sunfmin/gogen"
)

type API struct {
	Contributions struct {
		Html struct {
			Tags []*Component `json:"tags"`
		} `json:"html"`
	} `json:"contributions"`
}

type Component struct {
	Name       string       `json:"name"`
	Attributes []*Attribute `json:"attributes"`
	Slots      interface{}  `json:"slots"`
	//Mixins      []string        `json:"mixins"`
	//ScopedSlots json.RawMessage `json:"scopedSlots"`
	//Events json.RawMessage `json:"events"`
}

type Attribute struct {
	Name    string          `json:"name"`
	Type    interface{}     `json:"type"`
	Default json.RawMessage `json:"default"`
	Source  *string         `json:"source"`
	Value   *AttributeValue `json:"value"`
}

type AttributeValue struct {
	Kind string      `json:"kind"`
	Type interface{} `json:"type"`
}

var comp = flag.String("comp", "v-btn", "Vuetify Component Name")
var list = flag.String("list", "", "List Components")

func main() {
	flag.Parse()
	cb, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	bs := bytes.Split(cb, []byte("module.exports = "))
	jsonBody := bs[0]
	if len(bs) > 1 {
		jsonBody = bs[1]
	}
	//doc := string(cb)
	api := API{}
	err = json.Unmarshal(jsonBody, &api)
	if err != nil {
		panic(err)
	}

	listName := *list
	if len(listName) > 0 {
		for _, comp := range api.Contributions.Html.Tags {
			if strings.Index(comp.Name, listName) >= 0 {
				fmt.Println(comp.Name)
			}
		}
		return
	}

	compName := *comp
	if len(compName) == 0 {
		panic("vuetifyapi2go -comp=v-btn")
	}

	var compAPI *Component

	for _, comp := range api.Contributions.Html.Tags {
		if compName == comp.Name {
			compAPI = comp
			break
		}
	}

	if compAPI == nil {
		panic("component " + compName + " not exists")
	}
	compName = strcase.ToKebab(compName)

	constructorName := strcase.ToCamel(compName)
	builderName := fmt.Sprintf("%sBuilder", constructorName)

	propSnips := Snippets()

	for _, p := range compAPI.Attributes {
		propAttrName := strcase.ToKebab(p.Name)
		funcName := strcase.ToCamel(p.Name)

		funcParamTypeName := "string"
		var typ = p.Type
		if typ == nil {
			typ = p.Value.Type
		}

		switch pt := typ.(type) {
		case string:
			funcParamTypeName = jsToGoType(pt)
		case []string:
			if len(pt) > 0 {
				funcParamTypeName = jsToGoType(pt)
			}
		case []interface{}:
			if len(pt) > 0 {
				funcParamTypeName = jsToGoType(pt)
			}
		case nil:
			funcParamTypeName = "string"
		default:
			panic(fmt.Sprintf("%#+v", p))
		}

		propFunc := Func(
			"func (b *$VBtnBuilder) $Color(v $string) (r *$VBtnBuilder)",
			"$VBtnBuilder", builderName,
			"$color", propAttrName,
			"$Color", funcName,
			"$string", funcParamTypeName,
		)

		switch funcParamTypeName {
		case "function":
		case "bool", "int":
			propSnips.Append(
				propFunc.BodySnippet(
					`b.tag.Attr(":$color", fmt.Sprint(v))
					return b`,
					"$color", propAttrName,
				),
			)
		case "[]string", "interface{}":
			propSnips.Append(
				propFunc.BodySnippet(
					`b.tag.Attr(":$color", v)
					return b`,
					"$color", propAttrName,
				),
			)
		default:
			propSnips.Append(
				propFunc.BodySnippet(
					`b.tag.Attr("$color", v)
					return b`,
					"$color", propAttrName,
				),
			)
		}
	}

	consSnippet := Snippet(`
				func $VBtn() (r *$VBtnBuilder) {
					r = &$VBtnBuilder{
						tag: h.Tag("$v-btn"),
					}
					return
				}

				`, "$VBtn", constructorName, "$VBtnBuilder", builderName, "$v-btn", compName)

	if compAPI.Slots != nil {
		switch st := compAPI.Slots.(type) {
		case []interface{}:
			if len(st) > 0 {
				consSnippet = Snippet(`
				func $VBtn(children ...h.HTMLComponent) (r *$VBtnBuilder) {
					r = &$VBtnBuilder{
						tag: h.Tag("$v-btn").Children(children...),
					}
					return
				}

				`, "$VBtn", constructorName, "$VBtnBuilder", builderName, "$v-btn", compName)

			}
		default:
			panic(fmt.Sprintf("%#+v", compAPI.Slots))
		}
	}

	f := File("").Package("vuetify").Body(
		Imports(
			"context",
			"fmt",
		).Body(
			Snippet("\n"),
			ImportAs("h", "github.com/theplant/htmlgo"),
		),
		Struct(builderName).FieldsSnippet("tag *h.HTMLTagBuilder"),

		consSnippet,
		propSnips,
		Snippet(`

	func (b *$VBtnBuilder) SetAttr(k string, v interface{}) {
		b.tag.SetAttr(k, v)
	}

	func (b *$VBtnBuilder) Attr(vs ...interface{}) (r *$VBtnBuilder) {
		b.tag.Attr(vs...)
		return b
	}

	func (b *$VBtnBuilder) Children(children ...h.HTMLComponent) (r *$VBtnBuilder) {
		b.tag.Children(children...)
		return b
	}

	func (b *$VBtnBuilder) AppendChildren(children ...h.HTMLComponent) (r *$VBtnBuilder) {
		b.tag.AppendChildren(children...)
		return b
	}

	func (b *$VBtnBuilder) PrependChildren(children ...h.HTMLComponent) (r *$VBtnBuilder) {
		b.tag.PrependChildren(children...)
		return b
	}

	func (b *$VBtnBuilder) Class(names ...string) (r *$VBtnBuilder) {
		b.tag.Class(names...)
		return b
	}

	func (b *$VBtnBuilder) ClassIf(name string, add bool) (r *$VBtnBuilder) {
		b.tag.ClassIf(name, add)
		return b
	}

	func (b *$VBtnBuilder) On(name string, value string) (r *$VBtnBuilder) {
		b.tag.Attr(fmt.Sprintf("v-on:%s", name), value)
		return b
	}

	func (b *$VBtnBuilder) Bind(name string, value string) (r *$VBtnBuilder) {
		b.tag.Attr(fmt.Sprintf("v-bind:%s", name), value)
		return b
	}

	func (b *$VBtnBuilder) MarshalHTML(ctx context.Context) (r []byte, err error) {
		return b.tag.MarshalHTML(ctx)
	}
	`, "$VBtnBuilder", builderName),
	)
	f.MustFprint(os.Stdout, context.TODO())
}

func jsToGoType(jstype interface{}) (r string) {

	switch pt := jstype.(type) {
	case []string:
		for _, p := range pt {
			if v, found := findJsToGoType(p); found {
				return v
			}
		}
	case []interface{}:
		for _, p := range pt {
			if v, found := findJsToGoType(p); found {
				return v
			}
		}
	}
	r, _ = findJsToGoType(jstype)
	return
}

func findJsToGoType(jstype interface{}) (r string, found bool) {
	djstype := strings.ToLower(fmt.Sprintf("%s", jstype))
	if djstype == "string" {
		return "string", true
	}

	if djstype == "boolean" {
		return "bool", true
	}

	if djstype == "number" {
		return "int", true
	}

	if djstype == "array" {
		return "[]string", true
	}

	if strings.HasSuffix(djstype, "[]") {
		return "interface{}", false
	}

	if djstype == "any" || djstype == "object" {
		return "interface{}", true
	}

	return djstype, false
}
