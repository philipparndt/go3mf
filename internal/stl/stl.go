package stl

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Combiner combines multiple STL files
type Combiner struct{}

// NewCombiner creates a new STL Combiner
func NewCombiner() *Combiner {
	return &Combiner{}
}

// Combine combines multiple ASCII STL files into one
func (c *Combiner) Combine(inputFiles []string, outputFile string) error {
	out, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer out.Close()

	writer := bufio.NewWriter(out)
	defer writer.Flush()

	// Write combined header
	if _, err := writer.WriteString("solid combined\n"); err != nil {
		return fmt.Errorf("error writing header: %w", err)
	}

	// Combine all STL files
	for _, inputFile := range inputFiles {
		if err := c.appendSTL(writer, inputFile); err != nil {
			return fmt.Errorf("error processing %s: %w", inputFile, err)
		}
	}

	// Write footer
	if _, err := writer.WriteString("endsolid combined\n"); err != nil {
		return fmt.Errorf("error writing footer: %w", err)
	}

	return nil
}

// appendSTL appends an STL file's facets to the output
func (c *Combiner) appendSTL(writer *bufio.Writer, inputFile string) error {
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inFacet := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip solid/endsolid lines from input files
		if strings.HasPrefix(trimmed, "solid ") || strings.HasPrefix(trimmed, "endsolid") {
			continue
		}

		// Track facet sections
		if strings.HasPrefix(trimmed, "facet") {
			inFacet = true
		} else if strings.HasPrefix(trimmed, "endfacet") {
			inFacet = false
		}

		// Write all facet data
		if inFacet || strings.HasPrefix(trimmed, "facet") || strings.HasPrefix(trimmed, "endfacet") {
			if _, err := writer.WriteString(line + "\n"); err != nil {
				return fmt.Errorf("error writing data: %w", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}
