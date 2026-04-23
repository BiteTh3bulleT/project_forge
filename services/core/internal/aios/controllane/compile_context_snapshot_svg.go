package controllane

import (
	"fmt"
	"strings"
)

type snapshotRailLayout struct {
	Name  string
	Label string
	X     int
	Y     int
	W     int
	H     int
}

func renderCompiledContextSnapshotSVG(snapshot compiledContextSnapshot) string {
	width := 1200
	height := 820
	rails := []snapshotRailLayout{
		{Name: "constraints", Label: "Constraints", X: 60, Y: 70, W: 240, H: 620},
		{Name: "evidence", Label: "Evidence", X: 330, Y: 70, W: 240, H: 620},
		{Name: "hypotheses", Label: "Hypotheses", X: 630, Y: 70, W: 240, H: 620},
		{Name: "loops", Label: "Loops", X: 900, Y: 70, W: 240, H: 620},
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="Context snapshot card">`, width, height, width, height))
	b.WriteString(`<rect width="100%" height="100%" fill="#f7f1e3"/>`)
	b.WriteString(`<rect x="24" y="24" width="1152" height="772" rx="24" fill="#fffaf0" stroke="#3d352d" stroke-width="2"/>`)
	b.WriteString(fmt.Sprintf(`<text x="60" y="52" font-family="monospace" font-size="18" fill="#2f2821">Snapshot %s</text>`, svgText(snapshot.Header.SnapshotKind)))
	b.WriteString(fmt.Sprintf(`<text x="1140" y="52" text-anchor="end" font-family="monospace" font-size="14" fill="#5a5148">%s</text>`, svgText(snapshot.Header.Fingerprint)))

	for _, rail := range rails {
		b.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="18" fill="%s" stroke="#3d352d" stroke-width="2"/>`, rail.X, rail.Y, rail.W, rail.H, railFill(rail.Name)))
		b.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="monospace" font-size="18" fill="#2f2821">%s</text>`, rail.X+18, rail.Y+30, svgText(rail.Label)))
	}

	objectiveX := 600
	objectiveY := 410
	b.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="86" fill="#fee6b8" stroke="#3d352d" stroke-width="3"/>`, objectiveX, objectiveY))
	b.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-family="monospace" font-size="22" fill="#2f2821">Objective</text>`, objectiveX, objectiveY-18))
	b.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-family="monospace" font-size="14" fill="#2f2821">%s</text>`, objectiveX, objectiveY+12, svgText(truncateSnapshotText(snapshot.Graph.Objective.Label, 26))))
	b.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-family="monospace" font-size="12" fill="#5a5148">%s</text>`, objectiveX, objectiveY+34, svgText(truncateSnapshotText(snapshot.Graph.Objective.Detail, 28))))

	nodeByID := map[string]compiledContextSnapshotNode{}
	for _, node := range snapshot.Graph.Nodes {
		nodeByID[node.ID] = node
	}
	railByName := map[string]snapshotRailLayout{}
	for _, rail := range rails {
		railByName[rail.Name] = rail
	}
	nodePositions := map[string][2]int{
		snapshot.Graph.Objective.ID: {objectiveX, objectiveY},
	}
	for _, rail := range snapshot.Graph.Rails {
		layout, ok := railByName[rail.Name]
		if !ok {
			continue
		}
		for idx, nodeID := range rail.NodeIDs {
			nodePositions[nodeID] = [2]int{layout.X + layout.W/2, layout.Y + 82 + idx*96}
		}
	}

	for _, edge := range snapshot.Graph.Edges {
		src, okSrc := nodePositions[edge.SourceID]
		dst, okDst := nodePositions[edge.TargetID]
		if !okSrc || !okDst {
			continue
		}
		b.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="2" stroke-dasharray="%s"/>`, src[0], src[1], dst[0], dst[1], edgeStroke(edge.Type), edgeDash(edge.Type)))
	}

	for _, rail := range snapshot.Graph.Rails {
		for _, nodeID := range rail.NodeIDs {
			node, ok := nodeByID[nodeID]
			if !ok {
				continue
			}
			pos := nodePositions[nodeID]
			b.WriteString(renderCompiledSnapshotNode(node, pos[0], pos[1]))
		}
	}

	b.WriteString(fmt.Sprintf(`<text x="60" y="760" font-family="monospace" font-size="12" fill="#5a5148">counts c:%d e:%d h:%d l:%d</text>`, snapshot.Header.Counts.Constraints, snapshot.Header.Counts.Evidence, snapshot.Header.Counts.Hypotheses, snapshot.Header.Counts.Loops))
	if snapshot.Header.ParentSnapshotID != "" {
		b.WriteString(fmt.Sprintf(`<text x="1140" y="760" text-anchor="end" font-family="monospace" font-size="12" fill="#5a5148">parent %s</text>`, svgText(snapshot.Header.ParentSnapshotID)))
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func renderCompiledSnapshotNode(node compiledContextSnapshotNode, cx, cy int) string {
	x := cx - 90
	y := cy - 28
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="180" height="56" rx="12" fill="%s" stroke="#3d352d" stroke-width="2"/>`, x, y, nodeFill(node.Rail)))
	b.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="monospace" font-size="13" fill="#2f2821">%s</text>`, x+12, y+18, svgText(truncateSnapshotText(node.Label, 20))))
	b.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="monospace" font-size="11" fill="#5a5148">%s</text>`, x+12, y+34, svgText(truncateSnapshotText(node.Detail, 22))))
	markerX := x + 12
	for _, marker := range node.Markers {
		b.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="monospace" font-size="10" fill="#7b2d26">%s</text>`, markerX, y+48, svgText(snapshotMarkerGlyph(marker))))
		markerX += 34
	}
	return b.String()
}

func railFill(name string) string {
	switch name {
	case "constraints":
		return "#f8d9c4"
	case "evidence":
		return "#d6ecd2"
	case "hypotheses":
		return "#d9e8fb"
	case "loops":
		return "#f7d8dd"
	default:
		return "#ece7df"
	}
}

func nodeFill(name string) string {
	switch name {
	case "constraints":
		return "#fff1e7"
	case "evidence":
		return "#eef9ec"
	case "hypotheses":
		return "#eef5ff"
	case "loops":
		return "#fff0f2"
	default:
		return "#f7f3ed"
	}
}

func edgeStroke(edgeType string) string {
	switch edgeType {
	case "constrains":
		return "#915c2c"
	case "hypothesizes":
		return "#2a5f9f"
	case "tracks":
		return "#9c2f46"
	case "contradicts":
		return "#b22222"
	default:
		return "#356640"
	}
}

func edgeDash(edgeType string) string {
	switch edgeType {
	case "hypothesizes":
		return "6 4"
	case "contradicts":
		return "3 3"
	default:
		return ""
	}
}

func snapshotMarkerGlyph(marker string) string {
	switch {
	case strings.HasPrefix(marker, "salience:high"):
		return "S:H"
	case strings.HasPrefix(marker, "salience:medium"):
		return "S:M"
	case strings.HasPrefix(marker, "salience:low"):
		return "S:L"
	case strings.HasPrefix(marker, "confidence:"):
		return "C:" + strings.TrimPrefix(marker, "confidence:")
	case marker == "blocker":
		return "B"
	case marker == "conflict":
		return "X"
	default:
		return marker
	}
}

func svgText(in string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(in)
}
