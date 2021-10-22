package models

// ContextKey is a string type used in context.WithValue
type ContextKey string

func (c ContextKey) String() string {
	return string(c)
}
