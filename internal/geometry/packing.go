package geometry

import (
	"math"
	"sort"
)

// Rectangle represents a 2D rectangle for packing
type Rectangle struct {
	X, Y          float64 // Position
	Width, Height float64 // Dimensions
	ID            int     // Object identifier
}

// PackingResult represents the result of packing an object
type PackingResult struct {
	X, Y   float64
	ID     int
	Fits   bool
	Width  float64
	Height float64
}

// Packer implements a 2D bin packing algorithm
type Packer struct {
	margin float64
	nodes  []*packNode
}

type packNode struct {
	x, y          float64
	width, height float64
	used          bool
	right         *packNode
	down          *packNode
}

// NewPacker creates a new bin packer with the specified margin between objects
func NewPacker(margin float64) *Packer {
	return &Packer{
		margin: margin,
		nodes:  make([]*packNode, 0),
	}
}

// Pack arranges rectangles using a simple shelf packing algorithm
// Returns the positions for each object
func (p *Packer) Pack(objects []Rectangle) []PackingResult {
	if len(objects) == 0 {
		return []PackingResult{}
	}

	// Sort objects by height (descending) for better packing
	sorted := make([]Rectangle, len(objects))
	copy(sorted, objects)
	sort.Slice(sorted, func(i, j int) bool {
		// Sort by area first, then by height
		areaI := sorted[i].Width * sorted[i].Height
		areaJ := sorted[j].Width * sorted[j].Height
		if areaI != areaJ {
			return areaI > areaJ
		}
		return sorted[i].Height > sorted[j].Height
	})

	results := make([]PackingResult, len(sorted))

	// Use shelf packing algorithm
	currentX := 0.0
	currentY := 0.0
	rowHeight := 0.0
	maxWidth := 0.0

	for i, obj := range sorted {
		// If this object doesn't fit in current row, start new row
		if i > 0 && currentX > 0 {
			// Simple heuristic: if we've used significant width, start new row
			// This prevents extremely wide layouts
			if currentX+obj.Width > 300.0 { // 300mm max width heuristic
				currentX = 0.0
				currentY += rowHeight + p.margin
				rowHeight = 0.0
			}
		}

		results[i] = PackingResult{
			X:      currentX,
			Y:      currentY,
			ID:     obj.ID,
			Fits:   true,
			Width:  obj.Width,
			Height: obj.Height,
		}

		// Update position for next object
		currentX += obj.Width + p.margin
		if obj.Height > rowHeight {
			rowHeight = obj.Height
		}
		if currentX > maxWidth {
			maxWidth = currentX
		}
	}

	return results
}

// PackGrid arranges objects in a simple grid pattern
// This is a simpler alternative that's easier to understand
func (p *Packer) PackGrid(objects []Rectangle, maxColumns int) []PackingResult {
	if len(objects) == 0 {
		return []PackingResult{}
	}

	if maxColumns <= 0 {
		maxColumns = int(math.Ceil(math.Sqrt(float64(len(objects)))))
	}

	results := make([]PackingResult, len(objects))

	// Calculate column widths and row heights
	col := 0
	row := 0
	columnWidths := make([]float64, maxColumns)
	rowHeights := make(map[int]float64)

	// First pass: determine column widths and row heights
	for i, obj := range objects {
		c := i % maxColumns
		r := i / maxColumns

		if obj.Width > columnWidths[c] {
			columnWidths[c] = obj.Width
		}
		if obj.Height > rowHeights[r] {
			rowHeights[r] = obj.Height
		}
	}

	// Second pass: position objects
	for i, obj := range objects {
		col = i % maxColumns
		row = i / maxColumns

		// Calculate X position (sum of previous column widths)
		x := 0.0
		for c := 0; c < col; c++ {
			x += columnWidths[c] + p.margin
		}

		// Calculate Y position (sum of previous row heights)
		y := 0.0
		for r := 0; r < row; r++ {
			y += rowHeights[r] + p.margin
		}

		results[i] = PackingResult{
			X:      x,
			Y:      y,
			ID:     obj.ID,
			Fits:   true,
			Width:  obj.Width,
			Height: obj.Height,
		}
	}

	return results
}

// PackOptimal tries to find an efficient packing using shelf algorithm
func (p *Packer) PackOptimal(objects []Rectangle, maxBuildPlateWidth float64) []PackingResult {
	if len(objects) == 0 {
		return []PackingResult{}
	}

	// Sort objects by height (descending) for better packing
	sorted := make([]Rectangle, len(objects))
	copy(sorted, objects)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Height > sorted[j].Height
	})

	results := make([]PackingResult, len(sorted))

	// Shelf packing with build plate width constraint
	currentX := 0.0
	currentY := 0.0
	shelfHeight := 0.0

	for i, obj := range sorted {
		// Check if object fits in current shelf
		if currentX > 0 && currentX+obj.Width > maxBuildPlateWidth {
			// Move to next shelf
			currentX = 0.0
			currentY += shelfHeight + p.margin
			shelfHeight = 0.0
		}

		results[i] = PackingResult{
			X:      currentX,
			Y:      currentY,
			ID:     obj.ID,
			Fits:   true,
			Width:  obj.Width,
			Height: obj.Height,
		}

		// Update for next object
		currentX += obj.Width + p.margin
		if obj.Height > shelfHeight {
			shelfHeight = obj.Height
		}
	}

	return results
}
