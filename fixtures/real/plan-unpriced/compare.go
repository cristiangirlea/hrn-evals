// Package compare ranks the variants of an eval for the losses page.
package compare

import "strconv"

// Variant is one variant's line in an eval's results table.
type Variant struct {
	Name           string
	Cost           float64 // total, USD
	MedianCost     float64 // per attempt, USD
	Tokens         float64 // median context tokens
	CostIsEstimate bool    // hrn estimated Cost from list prices
}

// cheaper reports whether a is the cheaper variant of the two: lower total
// cost, with fewer median context tokens breaking equal costs.
func cheaper(a, b Variant) bool {
	if a.Cost != b.Cost {
		return a.Cost < b.Cost
	}
	return a.Tokens < b.Tokens
}

// cost formats one cost for a table cell, marking a non-zero cost as an estimate.
func cost(n float64) string {
	s := strconv.FormatFloat(n, 'f', -1, 64)
	if n != 0 {
		s += " (estimate)"
	}
	return s
}

// Losses lists the pairs where the cheaper variant passed fewer repetitions.
func Losses(variants []Variant, passed map[string]int) []string {
	var out []string
	for _, a := range variants {
		for _, b := range variants {
			if cheaper(a, b) && passed[a.Name] < passed[b.Name] {
				out = append(out, a.Name+" ("+cost(a.Cost)+") / "+b.Name+" ("+cost(b.Cost)+")")
			}
		}
	}
	return out
}
