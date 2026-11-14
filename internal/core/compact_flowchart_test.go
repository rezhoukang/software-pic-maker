package core

import "testing"

func TestCompactFlowchartUsesPositiveLabelForTheTrunk(t *testing.T) {
	d := Templates()[3]
	// Alternate relation order must not decide which branch becomes the trunk.
	d.Model.Relations[2], d.Model.Relations[3] = d.Model.Relations[3], d.Model.Relations[2]
	s := Generate(d)
	check, save, reject := sceneNode(s, "check"), sceneNode(s, "save"), sceneNode(s, "reject")
	if center(check).X != center(save).X || reject.X <= save.X+save.Width {
		t.Fatal("positive path must remain straight with the negative path to its right")
	}
	if s.Height > 560 {
		t.Fatal("the simple flowchart should fit a compact page", s.Height)
	}
	for _, e := range s.Edges {
		for i := 1; i < len(e.Points); i++ {
			for _, n := range s.Nodes {
				if n.ID == e.Source || n.ID == e.Target || n.Shape == "junction" {
					continue
				}
				if (routeBox{left: n.X, right: n.X + n.Width, top: n.Y, bottom: n.Y + n.Height}).crosses(e.Points[i-1], e.Points[i]) {
					t.Fatal("compact connector crosses a node", e.ID, n.ID)
				}
			}
		}
	}
}
