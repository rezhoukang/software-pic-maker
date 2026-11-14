package core

import (
	"reflect"
	"testing"
)

func saveFlowFixture(direction string) Diagram {
	return Diagram{Type: "flowchart", View: View{Direction: direction, Theme: "mono", Positions: map[string]Point{}}, Model: Model{
		Nodes:     []Item{{ID: "start", Name: "开始", Shape: "terminal"}, {ID: "snapshot", Name: "获取快照"}, {ID: "read", Name: "读取当前文件"}, {ID: "revision", Name: "版本一致？", Shape: "decision"}, {ID: "validate", Name: "图表有效？", Shape: "decision"}, {ID: "write", Name: "原子写入设计数据"}, {ID: "written", Name: "写入成功？", Shape: "decision"}, {ID: "update", Name: "更新保存状态"}, {ID: "recover", Name: "写入恢复副本"}, {ID: "error", Name: "保留编辑并提示错误"}, {ID: "end", Name: "结束", Shape: "terminal"}},
		Relations: []Relation{{ID: "f1", Source: "start", Target: "snapshot"}, {ID: "f2", Source: "snapshot", Target: "read"}, {ID: "f3", Source: "read", Target: "revision"}, {ID: "f4", Source: "revision", Target: "validate", Label: "是"}, {ID: "f5", Source: "revision", Target: "error", Label: "否"}, {ID: "f6", Source: "validate", Target: "write", Label: "是"}, {ID: "f7", Source: "validate", Target: "error", Label: "否"}, {ID: "f8", Source: "write", Target: "written"}, {ID: "f9", Source: "written", Target: "update", Label: "是"}, {ID: "f10", Source: "written", Target: "error", Label: "否"}, {ID: "f11", Source: "update", Target: "recover"}, {ID: "f12", Source: "recover", Target: "end"}, {ID: "f13", Source: "error", Target: "end"}},
	}}
}

func TestFlowchartSameSideBranchesNestWithoutCrossing(t *testing.T) {
	for _, direction := range []string{"TB", "LR"} {
		t.Run(direction, func(t *testing.T) {
			d := saveFlowFixture(direction)
			before := raw(t, d)
			s := Generate(d)
			if !reflect.DeepEqual(before, raw(t, d)) || !reflect.DeepEqual(s, Generate(d)) {
				t.Fatal("nondeterministic or mutated diagram")
			}
			branches := []Edge{}
			for _, e := range s.Edges {
				if e.ID == "f5" || e.ID == "f7" || e.ID == "f10" {
					branches = append(branches, e)
					if e.Target != "error" || e.Arrow != "open" {
						t.Fatal("branch semantics changed")
					}
					if len(e.Points) > 4 {
						t.Fatal("too many bends")
					}
				}
			}
			if len(branches) != 3 {
				t.Fatal("missing failure branches")
			}
			for i, a := range branches {
				for _, b := range branches[i+1:] {
					for j := 1; j < len(a.Points); j++ {
						for k := 1; k < len(b.Points); k++ {
							if architectureIntersects(a.Points[j-1], a.Points[j], b.Points[k-1], b.Points[k]) || architectureOverlap(a.Points[j-1], a.Points[j], b.Points[k-1], b.Points[k]) > .01 {
								t.Fatal("same-side branches cross or overlap", a.ID, b.ID, a.Points, b.Points)
							}
						}
					}
				}
			}
			assertXML(t, SVG(s))
			assertXML(t, DrawIO(s))
		})
	}
}
