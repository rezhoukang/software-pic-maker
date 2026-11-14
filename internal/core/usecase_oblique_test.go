package core

import (
	"math"
	"testing"
)

func TestUseCaseRelationshipsNeverHaveRightAngleBends(t *testing.T) {
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
			for i := 1; i < len(e.Points)-1; i++ {
				a, b, c := e.Points[i-1], e.Points[i], e.Points[i+1]
				dx, dy, ex, ey := b.X-a.X, b.Y-a.Y, c.X-b.X, c.Y-b.Y
				cosine := math.Abs(dx*ex+dy*ey) / (math.Hypot(dx, dy) * math.Hypot(ex, ey))
				if cosine < .001 {
					t.Fatal("right-angle use-case connector", e.ID, e.Points)
				}
			}
		}
	}
}
