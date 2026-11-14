package core

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
	"tuxu/shared"
)

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type Field struct {
	Name    string `json:"name"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}
type Item struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Fields  []Field  `json:"fields,omitempty"`
	Methods []string `json:"methods,omitempty"`
	Shape   string   `json:"shape,omitempty"`
	Role    string   `json:"role,omitempty"`
}
type Relation struct {
	ID                string `json:"id"`
	Source            string `json:"source"`
	Target            string `json:"target"`
	Label             string `json:"label,omitempty"`
	Kind              string `json:"kind,omitempty"`
	SourceCardinality string `json:"source_cardinality,omitempty"`
	TargetCardinality string `json:"target_cardinality,omitempty"`
}
type Message struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"`
	Kind   string `json:"kind,omitempty"`
}
type Model struct {
	Entities     []Item     `json:"entities,omitzero"`
	Actors       []Item     `json:"actors,omitzero"`
	UseCases     []Item     `json:"use_cases,omitzero"`
	Classes      []Item     `json:"classes,omitzero"`
	Participants []Item     `json:"participants,omitzero"`
	Nodes        []Item     `json:"nodes,omitzero"`
	Relations    []Relation `json:"relations,omitzero"`
	Messages     []Message  `json:"messages,omitzero"`
	SystemName   string     `json:"system_name,omitempty"`
}
type View struct {
	Theme     string           `json:"theme"`
	Direction string           `json:"direction"`
	Positions map[string]Point `json:"positions"`
}
type Diagram struct {
	SchemaVersion string `json:"schema_version"`
	ID            string `json:"id"`
	Type          string `json:"type"`
	Title         string `json:"title"`
	Description   string `json:"description,omitempty"`
	Model         Model  `json:"model"`
	View          View   `json:"view"`
}
type Project struct {
	SchemaVersion string    `json:"schema_version"`
	Name          string    `json:"name"`
	Diagrams      []Diagram `json:"diagrams"`
	UpdatedAt     string    `json:"updated_at,omitempty"`
}
type Issue struct {
	Path     string `json:"path"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}
type Validation struct {
	Valid  bool    `json:"valid"`
	Issues []Issue `json:"issues"`
}
type Node struct {
	ID      string   `json:"id"`
	X       float64  `json:"x"`
	Y       float64  `json:"y"`
	Width   float64  `json:"width"`
	Height  float64  `json:"height"`
	Shape   string   `json:"shape"`
	Title   string   `json:"title"`
	Lines   []string `json:"lines,omitempty"`
	Methods []string `json:"methods,omitempty"`
	Fill    string   `json:"fill"`
	Stroke  string   `json:"stroke"`
	Movable bool     `json:"movable"`
}
type Edge struct {
	ID                  string  `json:"id"`
	Source              string  `json:"source,omitempty"`
	Target              string  `json:"target,omitempty"`
	Points              []Point `json:"points"`
	Label               string  `json:"label,omitempty"`
	LabelPosition       *Point  `json:"label_position,omitempty"`
	SourceLabelPosition *Point  `json:"source_label_position,omitempty"`
	TargetLabelPosition *Point  `json:"target_label_position,omitempty"`
	SourceLabel         string  `json:"source_label,omitempty"`
	TargetLabel         string  `json:"target_label,omitempty"`
	Dashed              bool    `json:"dashed"`
	Arrow               string  `json:"arrow"`
}
type Scene struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Nodes  []Node  `json:"nodes"`
	Edges  []Edge  `json:"edges"`
	Type   string  `json:"type"`
	Title  string  `json:"title"`
	Theme  string  `json:"theme"`
}

var schemaOnce sync.Once
var resolvedSchema *jsonschema.Resolved
var schemaErr error

func ValidateJSON(data []byte) (Diagram, Validation) {
	v := Validation{Valid: true, Issues: []Issue{}}
	add := func(path, message string) {
		v.Valid = false
		v.Issues = append(v.Issues, Issue{path, message, "error"})
	}
	var generic any
	if err := json.Unmarshal(data, &generic); err != nil {
		add("$", "JSON 格式错误："+err.Error())
		return Diagram{}, v
	}
	schemaOnce.Do(func() {
		var s jsonschema.Schema
		schemaErr = json.Unmarshal(shared.DiagramSchema, &s)
		if schemaErr == nil {
			resolvedSchema, schemaErr = s.Resolve(nil)
		}
	})
	if schemaErr != nil {
		add("$", "Schema 加载失败："+schemaErr.Error())
		return Diagram{}, v
	}
	if err := resolvedSchema.Validate(generic); err != nil {
		add("$", "结构校验失败："+err.Error())
		return Diagram{}, v
	}
	var d Diagram
	if err := json.Unmarshal(data, &d); err != nil {
		add("$", err.Error())
		return d, v
	}
	seen := map[string]bool{}
	items := d.Items()
	for i, n := range items {
		if seen[n.ID] {
			add(fmt.Sprintf("model.items[%d].id", i), "ID 重复："+n.ID)
		}
		seen[n.ID] = true
	}
	ids := map[string]bool{}
	check := func(id, source, target, path string) {
		if ids[id] || seen[id] {
			add(path+".id", "ID 重复："+id)
		}
		ids[id] = true
		if !seen[source] {
			add(path+".source", "引用了不存在的节点："+source)
		}
		if !seen[target] {
			add(path+".target", "引用了不存在的节点："+target)
		}
	}
	for i, r := range d.Model.Relations {
		check(r.ID, r.Source, r.Target, fmt.Sprintf("model.relations[%d]", i))
	}
	for i, m := range d.Model.Messages {
		check(m.ID, m.Source, m.Target, fmt.Sprintf("model.messages[%d]", i))
	}
	if d.Type == "usecase" {
		actors := map[string]bool{}
		for _, a := range d.Model.Actors {
			actors[a.ID] = true
		}
		for i, r := range d.Model.Relations {
			p := fmt.Sprintf("model.relations[%d]", i)
			if r.Kind != "" && r.Kind != "association" && r.Kind != "include" && r.Kind != "extend" {
				add(p+".kind", "用例图关系仅支持 association、include、extend")
			}
			if r.Kind == "include" || r.Kind == "extend" {
				if r.Source == r.Target {
					add(p, "include / extend 不能连接同一个用例")
				}
				if actors[r.Source] || actors[r.Target] {
					add(p, "include / extend 必须连接两个用例")
				}
			} else if actors[r.Source] == actors[r.Target] {
				add(p, "association 必须连接一个参与者和一个用例")
			}
		}
	}
	if d.Type == "class" || d.Type == "structure" {
		adj := map[string][]string{}
		for _, r := range d.Model.Relations {
			if d.Type == "structure" || r.Kind == "inheritance" {
				adj[r.Source] = append(adj[r.Source], r.Target)
			}
		}
		state := map[string]int{}
		var cycle func(string) bool
		cycle = func(id string) bool {
			if state[id] == 1 {
				return true
			}
			if state[id] == 2 {
				return false
			}
			state[id] = 1
			for _, next := range adj[id] {
				if cycle(next) {
					return true
				}
			}
			state[id] = 2
			return false
		}
		for _, n := range items {
			if cycle(n.ID) {
				add("model.relations", "层级或继承关系不能形成循环")
				break
			}
		}
	}
	for _, e := range d.Model.Entities {
		fs := map[string]bool{}
		for _, f := range e.Fields {
			if fs[f.Name] {
				add("model.entities", "实体「"+e.Name+"」中字段重复："+f.Name)
			}
			fs[f.Name] = true
		}
	}
	connected := map[string]bool{}
	for _, r := range d.Model.Relations {
		connected[r.Source] = true
		connected[r.Target] = true
	}
	for _, m := range d.Model.Messages {
		connected[m.Source] = true
		connected[m.Target] = true
	}
	if len(items) > 1 {
		for _, n := range items {
			if !connected[n.ID] {
				v.Issues = append(v.Issues, Issue{"model", "「" + n.Name + "」暂未连接其他元素", "warning"})
			}
		}
	}
	return d, v
}

func (d Diagram) Items() []Item {
	switch d.Type {
	case "er", "table":
		return d.Model.Entities
	case "usecase":
		return append(append([]Item{}, d.Model.Actors...), d.Model.UseCases...)
	case "class":
		return d.Model.Classes
	case "sequence":
		return d.Model.Participants
	default:
		return d.Model.Nodes
	}
}
func ParseProject(data []byte) (Project, error) {
	var p Project
	var raw struct {
		Diagrams []json.RawMessage `json:"diagrams"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return p, err
	}
	for _, d := range raw.Diagrams {
		if _, v := ValidateJSON(d); !v.Valid {
			return p, fmt.Errorf("%s", v.Issues[0].Message)
		}
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return p, err
	}
	if p.SchemaVersion != "1.0" || strings.TrimSpace(p.Name) == "" || len(p.Diagrams) == 0 || len(p.Diagrams) > 100 {
		return p, fmt.Errorf("项目需包含版本 1.0、名称及 1～100 张图")
	}
	ids := map[string]bool{}
	for _, d := range p.Diagrams {
		if ids[d.ID] {
			return p, fmt.Errorf("图表 ID 重复：%s", d.ID)
		}
		ids[d.ID] = true
		b, _ := json.Marshal(d)
		_, v := ValidateJSON(b)
		if !v.Valid {
			return p, fmt.Errorf("%s：%s", d.Title, v.Issues[0].Message)
		}
	}
	return p, nil
}
func Templates() []Diagram { var ds []Diagram; _ = json.Unmarshal(shared.Library, &ds); return ds }
