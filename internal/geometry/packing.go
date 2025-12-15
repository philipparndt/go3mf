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

// PackCompact arranges objects as compactly as possible using a guillotine algorithm
// This algorithm recursively partitions the space to create a compact rectangular packing
// Result is a more balanced layout in both X and Y directions to reduce printer head travel time
func (p *Packer) PackCompact(objects []Rectangle) []PackingResult {
	if len(objects) == 0 {
		return []PackingResult{}
	}

	// Sort objects by height (descending), then by width for better packing
	sorted := make([]Rectangle, len(objects))
	copy(sorted, objects)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Height != sorted[j].Height {
			return sorted[i].Height > sorted[j].Height
		}
		return sorted[i].Width > sorted[j].Width
	})

	results := make([]PackingResult, len(sorted))

	// Find the maximum object dimensions to ensure we have enough space
	maxObjWidth := 0.0
	maxObjHeight := 0.0
	for _, obj := range sorted {
		if obj.Width > maxObjWidth {
			maxObjWidth = obj.Width
		}
		if obj.Height > maxObjHeight {
			maxObjHeight = obj.Height
		}
	}

	// Calculate total area to estimate optimal bin dimensions
	totalArea := 0.0
	for _, obj := range sorted {
		totalArea += (obj.Width + p.margin) * (obj.Height + p.margin)
	}

	// Estimate optimal width to create a more square layout
	// Ensure it's at least as wide as the widest object + margin
	optimalWidth := math.Sqrt(totalArea * 1.2) // 20% padding
	if optimalWidth < maxObjWidth+p.margin {
		optimalWidth = maxObjWidth + p.margin
	}
	if optimalWidth < 100 {
		optimalWidth = 100
	}
	// No upper limit - let it grow as needed

	// Track current layout bounds for fallback positioning
	currentMaxY := 0.0
	currentMaxX := 0.0

	// Create initial packing spaces with unlimited height
	spaces := []struct {
		x, y, width, height float64
	}{
		{0, 0, optimalWidth, 10000.0}, // Large height to allow vertical growth
	}

	// Pack each object into available spaces
	for i, obj := range sorted {
		packed := false

		// Try to fit object in existing spaces
		for spaceIdx, space := range spaces {
			if obj.Width+p.margin <= space.width && obj.Height+p.margin <= space.height {
				// Place object in this space
				results[i] = PackingResult{
					X:      space.x,
					Y:      space.y,
					ID:     obj.ID,
					Fits:   true,
					Width:  obj.Width,
					Height: obj.Height,
				}

				// Track bounds for fallback
				if space.x+obj.Width > currentMaxX {
					currentMaxX = space.x + obj.Width
				}
				if space.y+obj.Height > currentMaxY {
					currentMaxY = space.y + obj.Height
				}

				// Split the space using guillotine method
				objWidthWithMargin := obj.Width + p.margin
				objHeightWithMargin := obj.Height + p.margin

				// Create two new spaces: right and bottom
				newSpaces := []struct {
					x, y, width, height float64
				}{}

				// Add remaining space to the right
				if space.width > objWidthWithMargin {
					newSpaces = append(newSpaces, struct {
						x, y, width, height float64
					}{
						x:      space.x + objWidthWithMargin,
						y:      space.y,
						width:  space.width - objWidthWithMargin,
						height: space.height,
					})
				}

				// Add remaining space below
				if space.height > objHeightWithMargin {
					newSpaces = append(newSpaces, struct {
						x, y, width, height float64
					}{
						x:      space.x,
						y:      space.y + objHeightWithMargin,
						width:  objWidthWithMargin,
						height: space.height - objHeightWithMargin,
					})
				}

				// Remove used space and add new spaces
				spaces = append(spaces[:spaceIdx], spaces[spaceIdx+1:]...)
				spaces = append(spaces, newSpaces...)

				// Sort spaces to prioritize placement in lower-left corner (compact layout)
				sort.Slice(spaces, func(a, b int) bool {
					if spaces[a].y != spaces[b].y {
						return spaces[a].y < spaces[b].y
					}
					return spaces[a].x < spaces[b].x
				})

				packed = true
				break
			}
		}

		if !packed {
			// Fallback: create a new row at the bottom for objects that don't fit
			// Place the object at (0, currentMaxY + margin)
			fallbackY := currentMaxY + p.margin
			results[i] = PackingResult{
				X:      0,
				Y:      fallbackY,
				ID:     obj.ID,
				Fits:   true, // Still fits, just needed a new row
				Width:  obj.Width,
				Height: obj.Height,
			}

			// Update bounds
			if obj.Width > currentMaxX {
				currentMaxX = obj.Width
			}
			currentMaxY = fallbackY + obj.Height

			// Remove any existing spaces that would overlap with this fallback placement
			// This prevents future objects from being placed in the same position
			objWidthWithMargin := obj.Width + p.margin
			objHeightWithMargin := obj.Height + p.margin
			newSpaces := []struct {
				x, y, width, height float64
			}{}
			for _, space := range spaces {
				// Check if space overlaps with the fallback object's footprint
				spaceRight := space.x + space.width
				spaceBottom := space.y + space.height
				objRight := objWidthWithMargin
				objBottom := fallbackY + objHeightWithMargin

				// No overlap if space is completely to the right, left, above, or below
				if space.x >= objRight || spaceRight <= 0 || space.y >= objBottom || spaceBottom <= fallbackY {
					newSpaces = append(newSpaces, space)
				}
				// Space overlaps - don't keep it (or we could split it, but simpler to discard)
			}
			spaces = newSpaces

			// Add remaining space to the right of this object
			if optimalWidth > obj.Width+p.margin {
				spaces = append(spaces, struct {
					x, y, width, height float64
				}{
					x:      obj.Width + p.margin,
					y:      fallbackY,
					width:  optimalWidth - obj.Width - p.margin,
					height: obj.Height + p.margin,
				})
			}
		}
	}

	return results
}
