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

	if compName == "all" {
		for _, comp := range api.Contributions.Html.Tags {
			generateComponent(comp, strcase.ToCamel(comp.Name), true)
		}
		return
	}

	var compAPI *Component

	for _, comp := range api.Contributions.Html.Tags {
		if compName == comp.Name {
			compAPI = comp
			break
		}
	}

	generateComponent(compAPI, compName, false)
}

func goFileName(compName string) string {
	return compName[2:] + ".go"
}

func goFixFileName(compName string) string {
	return "fix-" + compName[2:] + ".go"
}

func funcInFix(builderName string, funcName string, fixContentString string) bool {
	funcSig, _ := Snippet("func (b *$VBtnBuilder) $Color(",
		"$VBtnBuilder", builderName,
		"$Color", funcName,
	).MarshalCode(context.TODO())

	if strings.Index(fixContentString, string(funcSig)) >= 0 {
		return true
	}

	return false
}

func generateComponent(compAPI *Component, compName string, toFile bool) {
	if compAPI == nil {
		panic("component " + compName + " not exists")
	}
	compName = strcase.ToKebab(compName)

	constructorName := strcase.ToCamel(compName)
	builderName := fmt.Sprintf("%sBuilder", constructorName)

	propSnips := Snippets()

	fixContent, _ := ioutil.ReadFile(goFixFileName(compName))
	fixContentString := string(fixContent)

	for _, p := range compAPI.Attributes {
		if strings.Index(p.Name, "(") >= 0 {
			continue
		}

		propAttrName := strcase.ToKebab(p.Name)
		funcName := strcase.ToCamel(p.Name)

		if funcInFix(builderName, funcName, fixContentString) {
			continue
		}

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
					`b.tag.Attr(":$color", h.JSONString(v))
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
				func $VBtn(children ...h.HTMLComponent) (r *$VBtnBuilder) {
					r = &$VBtnBuilder{
						tag: h.Tag("$v-btn").Children(children...),
					}
					return
				}

				`, "$VBtn", constructorName, "$VBtnBuilder", builderName, "$v-btn", compName)

	marshalSnippet := Snippet(`

	func (b *$VBtnBuilder) MarshalHTML(ctx context.Context) (r []byte, err error) {
		return b.tag.MarshalHTML(ctx)
	}
`, "$VBtnBuilder", builderName)

	imports := []string{"context", "fmt"}
	if funcInFix(builderName, "MarshalHTML", fixContentString) {
		marshalSnippet = Snippet("")
		imports = []string{"fmt"}
	}

	//if compAPI.Slots != nil {
	//	switch st := compAPI.Slots.(type) {
	//	case []interface{}:
	//		if len(st) > 0 {
	//			consSnippet = Snippet(`
	//			func $VBtn(children ...h.HTMLComponent) (r *$VBtnBuilder) {
	//				r = &$VBtnBuilder{
	//					tag: h.Tag("$v-btn").Children(children...),
	//				}
	//				return
	//			}
	//
	//			`, "$VBtn", constructorName, "$VBtnBuilder", builderName, "$v-btn", compName)
	//
	//		}
	//	default:
	//		panic(fmt.Sprintf("%#+v", compAPI.Slots))
	//	}
	//}

	var structCode Code = Struct(builderName).FieldsSnippet("tag *h.HTMLTagBuilder")
	if strings.Index(fixContentString,
		fmt.Sprintf("type %s struct", builderName)) >= 0 {
		structCode = Snippet("")
	}

	if strings.Index(fixContentString,
		fmt.Sprintf("func %s(", constructorName)) >= 0 {
		consSnippet = Snippet("")
	}

	f := File("").Package("vuetify").Body(
		Imports(
			imports...,
		).Body(
			Snippet("\n"),
			ImportAs("h", "github.com/theplant/htmlgo"),
		),
		structCode,
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

	`, "$VBtnBuilder", builderName),
		marshalSnippet,
	)

	output := os.Stdout
	if toFile {
		var err error
		output, err = os.OpenFile(goFileName(compName), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
		if err != nil {
			panic(err)
		}
		defer output.Close()
	}
	f.MustFprint(output, context.TODO())
}

func jsToGoType(jstype interface{}) (r string) {
	switch pt := jstype.(type) {
	case []interface{}:
		val := fmt.Sprint(pt)
		//fmt.Println(val)
		if val == `[boolean string number]` {
			return "interface{}"
		} else if val == `[string number]` {
			return "string"
		} else if val == `[string boolean]` {
			return "string"
		} else if strings.Index(val, "object") >= 0 {
			return "interface{}"
		} else if strings.Index(val, "number") >= 0 {
			return "int"
		} else if strings.Index(val, "boolean") >= 0 {
			return "bool"
		} else if strings.Index(val, "string") >= 0 {
			return "string"
		} else {
			return "interface{}"
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

	if djstype == "number" {
		return "int", true
	}

	if djstype == "boolean" {
		return "bool", true
	}

	if djstype == "array" {
		return "interface{}", true
	}

	if strings.HasSuffix(djstype, "[]") {
		return "interface{}", false
	}

	if djstype == "any" || djstype == "object" {
		return "interface{}", true
	}

	return "interface{}", false
}
