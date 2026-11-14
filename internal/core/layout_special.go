package core

import (
	"math"
)

func connectionCenter(n Node) Point {
	if n.Shape == "actor" {
		return Point{n.X + n.Width/2, n.Y + 48}
	}
	return center(n)
}

func layoutSequence(d Diagram, s *Scene, add func(Node), find func(string) Node) {
	width, step := 180.0, 250.0
	indices := map[string]int{}
	for i, p := range d.Model.Participants {
		width = math.Max(width, textWidth(p.Name, 180, 320))
		indices[p.ID] = i
	}
	step = width + 70
	for _, m := range d.Model.Messages {
		distance := math.Abs(float64(indices[m.Source] - indices[m.Target]))
		if distance > 0 {
			step = math.Max(step, (textWidth(m.Label, 150, 650)+48)/distance)
		}
	}
	lastRight := -40.0
	for i, p := range d.Model.Participants {
		add(Node{ID: p.ID, X: 64 + float64(i)*step, Y: 64, Width: width, Height: 58, Shape: "participant", Title: p.Name, Movable: true})
		n := &s.Nodes[len(s.Nodes)-1]
		// Sequence participants share a header row and never overlap horizontally.
		n.Y = 64
		n.X = math.Max(n.X, lastRight+64)
		lastRight = n.X + n.Width
	}
	base, gap := 178.0, 78.0
	bottom := base + float64(len(d.Model.Messages))*gap + 24
	for _, p := range d.Model.Participants {
		n := find(p.ID)
		x := center(n).X
		s.Edges = append(s.Edges, Edge{ID: "$lifeline:" + p.ID, Points: []Point{{x, n.Y + n.Height - 1}, {x, bottom}}, Dashed: true, Arrow: "none"})
	}
	type activation struct {
		id, caller    string
		start, end, x float64
		message       int
	}
	activations := []activation{}
	pending := []int{}
	for i, m := range d.Model.Messages {
		y := base + float64(i)*gap
		if m.Kind == "return" {
			for j := len(pending) - 1; j >= 0; j-- {
				a := &activations[pending[j]]
				if a.id == m.Source && a.caller == m.Target {
					a.end = y
					pending = append(pending[:j], pending[j+1:]...)
					break
				}
			}
			continue
		}
		start := y
		if m.Source == m.Target {
			start += 34
		}
		depth := 0
		for _, idx := range pending {
			if activations[idx].id == m.Target {
				depth++
			}
		}
		activations = append(activations, activation{id: m.Target, caller: m.Source, start: start, end: bottom - 20, x: center(find(m.Target)).X + float64(depth)*6, message: i})
		pending = append(pending, len(activations)-1)
	}
	for _, a := range activations {
		m := d.Model.Messages[a.message]
		add(Node{ID: "$activation:" + m.ID, X: a.x - 6, Y: a.start, Width: 12, Height: math.Max(34, a.end-a.start), Shape: "activation"})
	}
	activeX := func(id string, y float64, exclude int) (float64, bool) {
		x, ok := center(find(id)).X, false
		for _, a := range activations {
			if a.id == id && y >= a.start && y <= a.end && a.message != exclude {
				x, ok = a.x, true
			}
		}
		return x, ok
	}
	for i, m := range d.Model.Messages {
		a, b := find(m.Source), find(m.Target)
		y := base + float64(i)*gap
		ax, bx := center(a).X, center(b).X
		sourceX, sourceActive := activeX(m.Source, y, -1)
		targetX, targetActive := activeX(m.Target, y, -1)
		ax, bx = sourceX, targetX
		direction := 1.0
		if bx < ax {
			direction = -1
		}
		if sourceActive {
			ax += direction * 5
		}
		if targetActive {
			bx -= direction * 5
		}
		points := []Point{{ax, y}, {bx, y}}
		if m.Source == m.Target {
			prior, priorActive := activeX(m.Source, y, i)
			ax = prior
			if priorActive {
				ax += 5
			}
			target, targetActive := activeX(m.Target, y+34, -1)
			if targetActive {
				target += 5
			}
			loop := math.Max(ax, target) + math.Max(90, labelWidth(m.Label)+24)
			points = []Point{{ax, y}, {loop, y}, {loop, y + 34}, {target, y + 34}}
		}
		arrow := "filled"
		if m.Kind == "return" {
			arrow = "open"
		}
		s.Edges = append(s.Edges, Edge{ID: m.ID, Points: points, Label: m.Label, Arrow: arrow, Dashed: m.Kind == "return"})
	}
}
