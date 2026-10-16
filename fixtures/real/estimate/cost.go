// Package estimate formats eval costs for the losses page.
package estimate

// Variant is one variant's line in an eval's results table.
type Variant struct {
	Name           string
	Cost           float64 // total, USD
	CostIsEstimate bool    // hrn estimated Cost from list prices
}

// Cost formats n, one of v's costs, for a table cell.
func Cost(v Variant, n float64) string {
	return ""
}
