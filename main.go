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

type API map[string]*Component
type Component struct {
	Props       []Prop          `json:"props"`
	Mixins      []string        `json:"mixins"`
	Slots       interface{}     `json:"slots"`
	ScopedSlots json.RawMessage `json:"scopedSlots"`
	Events      json.RawMessage `json:"events"`
}

type Prop struct {
	Name    string          `json:"name"`
	Type    interface{}     `json:"type"`
	Default json.RawMessage `json:"default"`
	Source  *string         `json:"source"`
}

var comp = flag.String("comp", "v-btn", "Vuetify Component Name")

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
	api := make(API)
	err = json.Unmarshal(jsonBody, &api)
	if err != nil {
		panic(err)
	}

	compName := *comp
	if len(compName) == 0 {
		panic("vuetifyapi2go -comp=v-btn")
	}

	constructorName := strcase.ToCamel(compName)
	builderName := fmt.Sprintf("%sBuilder", constructorName)
	compAPI := api[compName]
	if compAPI == nil {
		panic("component " + compName + " not exists")
	}

	propSnips := Snippets()

	for _, p := range compAPI.Props {
		propAttrName := strcase.ToKebab(p.Name)
		funcName := strcase.ToCamel(p.Name)

		funcParamTypeName := "string"

		switch pt := p.Type.(type) {
		case string:
			funcParamTypeName = jsToGoType(pt)
		case []string:
			if len(pt) > 0 {
				funcParamTypeName = jsToGoType(pt[0])
			}
		case []interface{}:
			if len(pt) > 0 {
				funcParamTypeName = jsToGoType(pt[0])
			}
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
					`b.tag.Attr(":$color", fmt.Sprint(v))`,
					"$color", propAttrName,
				),
			)
		case "[]string", "interface{}":
			propSnips.Append(
				propFunc.BodySnippet(
					`b.tag.Attr(":$color", v)`,
					"$color", propAttrName,
				),
			)
		default:
			propSnips.Append(
				propFunc.BodySnippet(
					`b.tag.Attr("$color", v)`,
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
				}`, "$VBtn", constructorName, "$VBtnBuilder", builderName, "$v-btn", compName)

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
				}`, "$VBtn", constructorName, "$VBtnBuilder", builderName, "$v-btn", compName)

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
			ImportAs("h", "github.com/theplant/htmlgo"),
		),
		Struct(builderName).FieldsSnippet("tag *h.HTMLTagBuilder"),

		consSnippet,
		propSnips,
		Snippet(`
	func (b *$VBtnBuilder) MarshalHTML(ctx context.Context) (r []byte, err error) {
		return b.tag.MarshalHTML(ctx)
	}`, "$VBtnBuilder", builderName),
	)
	err = f.Fprint(os.Stdout, context.TODO())
	if err != nil {
		panic(err)
	}
}

func jsToGoType(jstype interface{}) string {
	djstype := strings.ToLower(fmt.Sprintf("%s", jstype))
	if djstype == "string" {
		return "string"
	}

	if djstype == "boolean" {
		return "bool"
	}

	if djstype == "number" {
		return "int"
	}

	if djstype == "array" {
		return "[]string"
	}

	if djstype == "any" {
		return "interface{}"
	}

	return djstype
}
