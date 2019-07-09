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

		switch funcParamTypeName {
		case "function", "any":
		case "bool", "int":
			propSnips.Append(
				Snippet(`
						func (b *$VBtnBuilder) $Color(v $string) (r *$VBtnBuilder) {
							b.tag.Attr(":$color", fmt.Sprint(v))
							return b
						}`, "$VBtnBuilder", builderName,
					"$color", propAttrName,
					"$Color", funcName,
					"$string", funcParamTypeName),
			)
		default:
			propSnips.Append(
				Snippet(`
						func (b *$VBtnBuilder) $Color(v $string) (r *$VBtnBuilder) {
							b.tag.Attr("$color", v)
							return b
						}`, "$VBtnBuilder", builderName,
					"$color", propAttrName,
					"$Color", funcName,
					"$string", funcParamTypeName),
			)
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

		Snippet(`
				func $VBtn() (r *$VBtnBuilder) {
					r = &VBtnBuilder{
						tag: h.Tag("$v-btn"),
					}
					return
				}`, "$VBtn", constructorName, "$VBtnBuilder", builderName, "$v-btn", compName),
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
		return "interface{}"
	}

	return djstype
}
