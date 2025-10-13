package testutil

import (
	"fmt"
	"strings"
)

const (
	tileSizePx  = 40 // SVG pixels per tile
	unitRadius  = 12
	marginPx    = 50
	legendHeight = 60
)

// RenderScenarioSVG generates an SVG diagram of the scenario
func RenderScenarioSVG(scenario *TestScenario, mapData any) (string, error) {
	// For now, we'll use basic map info
	// TODO: Pass actual MapData when available
	mapWidth := 20  // Default for test maps
	mapHeight := 15

	svgWidth := mapWidth*tileSizePx + 2*marginPx
	svgHeight := mapHeight*tileSizePx + 2*marginPx + legendHeight

	var sb strings.Builder

	// SVG header
	sb.WriteString(fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">`, svgWidth, svgHeight))
	sb.WriteString("\n")

	// Styles
	sb.WriteString(`<defs>`)
	sb.WriteString(`<style>`)
	sb.WriteString(`.grid { stroke: #ddd; stroke-width: 1; fill: none; }`)
	sb.WriteString(`.tile { fill: #f9f9f9; stroke: #ddd; stroke-width: 1; }`)
	sb.WriteString(`.rock { fill: #888; stroke: #555; stroke-width: 2; }`)
	sb.WriteString(`.unit-start { fill: #4488ff; stroke: #003366; stroke-width: 2; }`)
	sb.WriteString(`.unit-end { fill: #44ff44; stroke: #006600; stroke-width: 2; }`)
	sb.WriteString(`.unit-label { fill: white; font-family: Arial; font-size: 14px; font-weight: bold; text-anchor: middle; dominant-baseline: middle; }`)
	sb.WriteString(`.path { stroke: #ff8800; stroke-width: 3; stroke-dasharray: 8,4; fill: none; marker-end: url(#arrowhead); }`)
	sb.WriteString(`.legend-text { font-family: Arial; font-size: 12px; fill: #333; }`)
	sb.WriteString(`.title-text { font-family: Arial; font-size: 16px; font-weight: bold; fill: #333; }`)
	sb.WriteString(`</style>`)

	// Arrow marker for paths
	sb.WriteString(`<marker id="arrowhead" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto">`)
	sb.WriteString(`<polygon points="0 0, 10 3, 0 6" fill="#ff8800"/>`)
	sb.WriteString(`</marker>`)
	sb.WriteString(`</defs>`)
	sb.WriteString("\n")

	// Background
	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" fill="#ffffff"/>`, svgWidth, svgHeight))
	sb.WriteString("\n")

	// Title
	titleY := 25
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="title-text">%s</text>`,
		svgWidth/2, titleY, escapeXML(scenario.Name)))
	sb.WriteString("\n")

	// Grid
	gridOffsetX := marginPx
	gridOffsetY := marginPx + 20 // Extra space for title

	// Draw grid tiles
	for y := 0; y < mapHeight; y++ {
		for x := 0; x < mapWidth; x++ {
			px := gridOffsetX + x*tileSizePx
			py := gridOffsetY + y*tileSizePx
			sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" class="tile"/>`,
				px, py, tileSizePx, tileSizePx))
			sb.WriteString("\n")
		}
	}

	// TODO: Draw terrain (rocks) from mapData
	// For now, placeholder rocks
	// This would come from the actual map file

	// Draw expected paths (if any visual annotations)
	if scenario.Visual != nil {
		for _, ann := range scenario.Visual.Annotations {
			if ann.Type == "arrow" && ann.From != nil && ann.To != nil {
				fromPx := tileToPixel(ann.From[0], ann.From[1], gridOffsetX, gridOffsetY)
				toPx := tileToPixel(ann.To[0], ann.To[1], gridOffsetX, gridOffsetY)
				sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" class="path"/>`,
					fromPx.x, fromPx.y, toPx.x, toPx.y))
				sb.WriteString("\n")
			}
		}
	}

	// Draw initial unit positions (blue circles)
	for _, unit := range scenario.Setup.Units {
		px := tileToPixel(unit.Position[0], unit.Position[1], gridOffsetX, gridOffsetY)
		sb.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="%d" class="unit-start"/>`,
			px.x, px.y, unitRadius))
		sb.WriteString("\n")

		label := unit.Label
		if label == "" {
			label = unit.ID
		}
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="unit-label">%s</text>`,
			px.x, px.y, escapeXML(label)))
		sb.WriteString("\n")
	}

	// Draw expected final positions (green circles)
	for _, expected := range scenario.Expectations.FinalState.Units {
		var pos [2]int
		if expected.Position != nil {
			pos = *expected.Position
		} else if expected.PositionNear != nil {
			pos = *expected.PositionNear
		} else {
			continue // No position specified
		}

		px := tileToPixel(pos[0], pos[1], gridOffsetX, gridOffsetY)
		sb.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="%d" class="unit-end"/>`,
			px.x, px.y, unitRadius))
		sb.WriteString("\n")

		label := expected.Label
		if label == "" {
			// Find original unit to get label
			setupUnit := scenario.GetUnitByID(expected.ID)
			if setupUnit != nil && setupUnit.Label != "" {
				label = setupUnit.Label + "'"
			} else {
				label = expected.ID
			}
		}
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="unit-label">%s</text>`,
			px.x, px.y, escapeXML(label)))
		sb.WriteString("\n")
	}

	// Legend
	legendY := svgHeight - legendHeight + 10
	sb.WriteString(fmt.Sprintf(`<rect x="0" y="%d" width="%d" height="%d" fill="#f0f0f0" stroke="#ccc"/>`,
		svgHeight-legendHeight, svgWidth, legendHeight))
	sb.WriteString("\n")

	// Legend items
	legendX := 20
	legendItemY := legendY + 20

	// Initial position
	sb.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="8" class="unit-start"/>`, legendX, legendItemY))
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="legend-text">Initial Position</text>`,
		legendX+15, legendItemY+4))

	// Expected final position
	legendX += 150
	sb.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="8" class="unit-end"/>`, legendX, legendItemY))
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="legend-text">Expected Final</text>`,
		legendX+15, legendItemY+4))

	// Obstacle
	legendX += 150
	sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="16" height="16" class="rock"/>`, legendX-8, legendItemY-8))
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="legend-text">Obstacle</text>`,
		legendX+15, legendItemY+4))

	// Expected path
	legendX += 120
	sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" class="path"/>`, legendX-10, legendItemY, legendX+10, legendItemY))
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="legend-text">Expected Path</text>`,
		legendX+20, legendItemY+4))

	// Description (if provided)
	if scenario.Description != "" {
		descY := legendItemY + 25
		sb.WriteString(fmt.Sprintf(`<text x="20" y="%d" class="legend-text" style="font-style: italic;">%s</text>`,
			descY, escapeXML(scenario.Description)))
	}

	sb.WriteString(`</svg>`)

	return sb.String(), nil
}

// pixel represents an SVG pixel coordinate
type pixel struct {
	x, y int
}

// tileToPixel converts tile coordinates to SVG pixel coordinates (center of tile)
func tileToPixel(tileX, tileY, offsetX, offsetY int) pixel {
	return pixel{
		x: offsetX + tileX*tileSizePx + tileSizePx/2,
		y: offsetY + tileY*tileSizePx + tileSizePx/2,
	}
}

// escapeXML escapes special XML characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
