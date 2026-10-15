# Validate

Validate returns an error for the first problem it finds in a matrix, in this
order, and nil when there is none:

1. An empty matrix: `matrix is empty`.
2. A variant whose DisplayName is empty (no name, backend, model or effort):
   `variant N has no name and no backend`, with N counting from 1.
3. Two variants with the same DisplayName: `duplicate variant "NAME"`, naming
   the second one's DisplayName.

The backend is always the first part of a derived name, even when it is empty:
a variant with only the model m1 is called `-m1`.
