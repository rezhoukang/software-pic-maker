package core

import (
	"fmt"
	"math"
	"testing"
)

func sharedActorUseCases() Diagram {
	d := Templates()[2]
	d.Model.Actors = []Item{{ID: "reader", Name: "读者"}, {ID: "reader2", Name: "参与者12"}, {ID: "actor3", Name: "参与者3"}, {ID: "actor4", Name: "参与者4"}}
	relations := []Relation{}
	for _, r := range d.Model.Relations {
		if r.Source != "admin" && r.Target != "admin" {
			relations = append(relations, r)
		}
	}
	d.Model.Relations = append(relations, Relation{ID: "shared", Source: "reader2", Target: "borrow"})
	for i := len(d.Model.UseCases); i < 22; i++ {
		d.Model.UseCases = append(d.Model.UseCases, Item{ID: fmt.Sprintf("case%d", i+1), Name: fmt.Sprintf("用例%d", i+1)})
	}
	return d
}

func TestSharedUseCaseActorsStayNearTheirCasesOnTheSameSide(t *testing.T) {
	s := Generate(sharedActorUseCases())
	boundary := sceneNode(s, "$boundary")
	a, b := sceneNode(s, "reader"), sceneNode(s, "reader2")
	if a.X >= boundary.X || b.X >= boundary.X {
		t.Fatal("actors of the same small group should share its near side", a, b)
	}
	if math.Abs(a.Y-b.Y) < actorNodeHeight || math.Abs(a.Y-b.Y) > actorNodeHeight+40 {
		t.Fatal("related actors should form a compact, non-overlapping stack", a, b)
	}
	assertUseCaseEdgesAvoidNodes(t, s)
	for _, e := range s.Edges {
		if (e.ID == "u1" || e.ID == "u2" || e.ID == "u3") && len(e.Points) != 2 {
			t.Fatal("reader associations should keep their direct fan", e.ID, e.Points)
		}
		if e.ID != "shared" {
			continue
		}
		length := 0.0
		for i := 1; i < len(e.Points); i++ {
			length += math.Hypot(e.Points[i].X-e.Points[i-1].X, e.Points[i].Y-e.Points[i-1].Y)
		}
		if length > 350 {
			t.Fatal("association still crosses the whole system", length)
		}
	}
}
