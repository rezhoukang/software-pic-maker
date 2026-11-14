package core

import "testing"

func TestUseCaseRelationshipsAreAlwaysSingleStraightSegments(t *testing.T) {
	moved := sharedActorUseCases()
	moved.View.Positions["reader"] = Point{55, 80}
	blocked := Templates()[2]
	blocked.Model.UseCases = append(blocked.Model.UseCases, Item{ID: "blocker", Name: "中间用例"})
	blocked.View.Positions["eligibility"] = Point{1200, 244}
	blocked.View.Positions["blocker"] = Point{650, 244}
	for _, d := range []Diagram{Templates()[2], sharedActorUseCases(), populatedUseCases(), moved, blocked} {
		s := Generate(d)
		assertUseCaseEdgesAvoidNodes(t, s)
		for _, e := range s.Edges {
			if len(e.Points) != 2 {
				t.Fatal("use-case connector must never bend", e.ID, e.Points)
			}
		}
	}
}
