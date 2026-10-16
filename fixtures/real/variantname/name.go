// Package variantname names the variants of an eval matrix and checks a
// matrix before an eval runs, as hrn does.
package variantname

// Variant is one entry of an eval spec's matrix.
type Variant struct {
	Name    string `json:"name,omitempty"`
	Backend string `json:"backend,omitempty"`
	Model   string `json:"model,omitempty"`
	Effort  string `json:"effort,omitempty"`
}

// DisplayName is the name results tables use for v.
func (v Variant) DisplayName() string {
	return v.Name
}

// Validate checks a matrix before an eval runs.
func Validate(matrix []Variant) error {
	return nil
}
