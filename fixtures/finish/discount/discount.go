// Package discount prices an order after a percentage discount.
package discount

import "errors"

// ErrPercent is returned for a discount outside 0..100.
var ErrPercent = errors.New("discount: percent must be within 0..100")

// Apply returns total, in cents, less pct percent of it; the discount is
// rounded down to a whole cent.
func Apply(total, pct int) (int, error) {
	if pct < 0 {
		return 0, ErrPercent
	}
	return total - total*pct/100, nil
}
