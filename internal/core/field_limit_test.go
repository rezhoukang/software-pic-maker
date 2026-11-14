package core

import (
	"fmt"
	"strings"
	"testing"
)

func TestFiftyFieldsValidateRenderAndExport(t *testing.T) {
	for _, index := range []int{0, 1, 5} {
		d := Templates()[index]
		item := &d.Model.Entities
		if d.Type == "class" {
			item = &d.Model.Classes
		}
		(*item)[0].Fields = nil
		for i := 0; i < 50; i++ {
			(*item)[0].Fields = append((*item)[0].Fields, Field{Name: fmt.Sprintf("field_%d", i+1)})
		}
		t.Run(d.Type, func(t *testing.T) {
			result := Render(string(raw(t, d)))
			if !result.Validation.Valid {
				t.Fatal(result.Validation.Issues)
			}
			if !strings.Contains(result.SVG, "field_50") {
				t.Fatal("last field is missing")
			}
			assertXML(t, result.SVG)
			exported, err := Export(string(raw(t, d)), "drawio")
			if err != nil || !strings.Contains(exported, "field_50") {
				t.Fatal("last field is missing in export", err)
			}
			assertXML(t, exported)
			(*item)[0].Fields = append((*item)[0].Fields, Field{Name: "field_51"})
			if _, validation := ValidateJSON(raw(t, d)); validation.Valid {
				t.Fatal("51 fields should exceed the limit")
			}
		})
	}
}
