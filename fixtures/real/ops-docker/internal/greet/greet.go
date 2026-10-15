// Package greet builds greetings.
package greet

// Hello greets name, or the world when name is empty.
func Hello(name string) string {
	if name == "" {
		name = "world"
	}
	return "hello, " + name
}
