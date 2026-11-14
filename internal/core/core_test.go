package core

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func raw(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func assertXML(t *testing.T, s string) {
	t.Helper()
	d := xml.NewDecoder(strings.NewReader(s))
	for {
		_, e := d.Token()
		if e == io.EOF {
			return
		}
		if e != nil {
			t.Fatal(e)
		}
	}
}
func TestAllTemplates(t *testing.T) {
	ds := Templates()
	if len(ds) != 8 {
		t.Fatal(len(ds))
	}
	for _, d := range ds {
		t.Run(d.Type, func(t *testing.T) {
			b := raw(t, d)
			_, v := ValidateJSON(b)
			if !v.Valid {
				t.Fatal(v.Issues)
			}
			r := Render(string(b))
			if len(r.Scene.Nodes) == 0 || r.Scene.Width < 1 {
				t.Fatal("empty scene")
			}
			assertXML(t, r.SVG)
			for _, svg := range r.NodeSVGs {
				assertXML(t, svg)
			}
			dx, err := Export(string(b), "drawio")
			if err != nil {
				t.Fatal(err)
			}
			assertXML(t, dx)
			// All draw.io cell IDs must be unique, and all connections must resolve.
			var doc struct {
				Diagram struct {
					Graph struct {
						Root struct {
							Cells []struct {
								ID     string `xml:"id,attr"`
								Source string `xml:"source,attr"`
								Target string `xml:"target,attr"`
								Parent string `xml:"parent,attr"`
							} `xml:"mxCell"`
						} `xml:"root"`
					} `xml:"mxGraphModel"`
				} `xml:"diagram"`
			}
			if err := xml.Unmarshal([]byte(dx), &doc); err != nil {
				t.Fatal(err)
			}
			ids := map[string]bool{}
			for _, c := range doc.Diagram.Graph.Root.Cells {
				if ids[c.ID] {
					t.Fatalf("duplicate cell %s", c.ID)
				}
				ids[c.ID] = true
			}
			for _, c := range doc.Diagram.Graph.Root.Cells {
				for _, id := range []string{c.Source, c.Target, c.Parent} {
					if id != "" && !ids[id] {
						t.Fatalf("missing cell %s", id)
					}
				}
			}
			exported, err := Export(string(b), "json")
			if err != nil {
				t.Fatal(err)
			}
			if _, v := ValidateJSON([]byte(exported)); !v.Valid {
				t.Fatal(v)
			}
		})
	}
}
func TestProjectRoundTripAndEmptyRelations(t *testing.T) {
	d := Templates()[0]
	d.Model.Relations = []Relation{}
	p := Project{SchemaVersion: "1.0", Name: "空关系项目", Diagrams: []Diagram{d}}
	path := filepath.Join(t.TempDir(), "项目.tuxu.json")
	for i := 0; i < 2; i++ {
		if err := AtomicWrite(path, raw(t, p)); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		p, err = ParseProject(b)
		if err != nil {
			t.Fatal(err)
		}
	}
	if p.Diagrams[0].Model.Relations == nil {
		t.Fatal("empty required array lost")
	}
}
func TestRejectInvalidModels(t *testing.T) {
	tests := []struct {
		name   string
		index  int
		change func(*Diagram)
	}{
		{"duplicate ID", 0, func(d *Diagram) { d.Model.Entities[1].ID = d.Model.Entities[0].ID }},
		{"missing reference", 0, func(d *Diagram) { d.Model.Relations[0].Target = "missing" }},
		{"duplicate field", 0, func(d *Diagram) {
			d.Model.Entities[0].Fields = append(d.Model.Entities[0].Fields, d.Model.Entities[0].Fields[0])
		}},
		{"actor include", 2, func(d *Diagram) { d.Model.Relations[0].Kind = "include" }},
		{"hierarchy cycle", 7, func(d *Diagram) {
			d.Model.Relations = append(d.Model.Relations, Relation{ID: "cycle", Source: "readers", Target: "system"})
		}},
		{"inheritance cycle", 5, func(d *Diagram) {
			d.Model.Relations = []Relation{{ID: "a", Source: "book", Target: "loan", Kind: "inheritance"}, {ID: "b", Source: "loan", Target: "book", Kind: "inheritance"}}
		}},
		{"invalid coordinates", 0, func(d *Diagram) { d.View.Positions["reader"] = Point{-1, 0} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Templates()[tt.index]
			tt.change(&d)
			if _, v := ValidateJSON(raw(t, d)); v.Valid {
				t.Fatal("accepted invalid diagram")
			}
		})
	}
}
func TestPositionStabilityAndEscaping(t *testing.T) {
	d := Templates()[0]
	d.View.Positions["reader"] = Point{0, 0}
	d.Model.Entities[0].Name = "<读者> & \"用户\""
	for i := 0; i < 3; i++ {
		s := Generate(d)
		for _, n := range s.Nodes {
			if n.ID == "reader" {
				if n.X != 0 || n.Y != 0 {
					t.Fatal("position shifted", n)
				}
				d.View.Positions[n.ID] = Point{n.X, n.Y}
			}
		}
		assertXML(t, SVG(s))
		assertXML(t, DrawIO(s))
	}
}
func TestGeneratedIDsDoNotCollide(t *testing.T) {
	d := Templates()[0]
	d.Model.Entities = append(d.Model.Entities, Item{ID: "reader-attr-0", Name: "合法用户 ID"})
	s := Generate(d)
	ids := map[string]bool{}
	for _, n := range s.Nodes {
		if ids[n.ID] {
			t.Fatal(n.ID)
		}
		ids[n.ID] = true
	}
	for _, e := range s.Edges {
		if ids[e.ID] {
			t.Fatal(e.ID)
		}
		ids[e.ID] = true
	}
}

func TestERManyAttributesDoNotOverlap(t *testing.T) {
	d := Templates()[0]
	for i := range d.Model.Entities {
		d.Model.Entities[i].Fields = []Field{}
		for j := 0; j < 30; j++ {
			d.Model.Entities[i].Fields = append(d.Model.Entities[i].Fields, Field{Name: fmt.Sprintf("比较长的业务属性名称_%d", j)})
		}
	}
	s := Generate(d)
	var attrs []Node
	for _, n := range s.Nodes {
		if strings.Contains(n.Shape, "ttribute") {
			attrs = append(attrs, n)
		}
	}
	for i, a := range attrs {
		for _, b := range attrs[i+1:] {
			if a.X < b.X+b.Width && a.X+a.Width > b.X && a.Y < b.Y+b.Height && a.Y+a.Height > b.Y {
				t.Fatalf("overlap: %s and %s", a.ID, b.ID)
			}
		}
	}
}

func TestUseCaseAssociationsDoNotCrossUnrelatedCases(t *testing.T) {
	s := Generate(Templates()[2])
	for _, e := range s.Edges {
		for _, n := range s.Nodes {
			if n.Shape != "ellipse" || n.ID == e.Source || n.ID == e.Target {
				continue
			}
			for step := 1; step < 100; step++ {
				p := pointAt(e.Points, float64(step)/100)
				x := (p.X - n.X - n.Width/2) / (n.Width / 2)
				y := (p.Y - n.Y - n.Height/2) / (n.Height / 2)
				if x*x+y*y < 1 {
					t.Fatalf("%s crosses %s", e.ID, n.ID)
				}
			}
		}
	}
}
