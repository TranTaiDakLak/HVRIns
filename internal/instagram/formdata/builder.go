// Package formdata — Build Facebook API form data and header collections
// Mapping từ C#: FacebookApiFormDataBuilder + FacebookApiHeaderCollectionBuilder
//
// Planned content:
//   - FormDataBuilder.Add/Build() → build URL-encoded or multipart form bodies
//   - HeaderBuilder.Set/Build() → build HTTP header map for Facebook API requests
//   - FormProp models for structured form field definitions
//
// TODO: Port from C# FacebookApiFormDataBuilder and related classes
package formdata

// FormProp describes a single form field with its name, value, and metadata.
// Mapping từ C#: FacebookRequestFormDataPropModel
type FormProp struct {
	Name     string
	Value    string
	Encode   bool // URL-encode the value
	Required bool
}

// Builder accumulates form fields and builds URL-encoded or multipart bodies.
// Mapping từ C#: FacebookApiFormDataBuilder
type Builder struct {
	props []FormProp
}

// Add appends a field to the builder.
func (b *Builder) Add(name, value string) *Builder {
	b.props = append(b.props, FormProp{Name: name, Value: value, Encode: true})
	return b
}

// AddRaw appends a field without URL-encoding its value.
func (b *Builder) AddRaw(name, value string) *Builder {
	b.props = append(b.props, FormProp{Name: name, Value: value, Encode: false})
	return b
}

// Build returns the URL-encoded form body string.
// TODO: full implementation with proper URL encoding and ordering
func (b *Builder) Build() string {
	// Placeholder — TODO: implement proper URL encoding
	return ""
}

// HeaderBuilder accumulates HTTP headers for Facebook API requests.
// Mapping từ C#: FacebookApiHeaderCollectionBuilder
type HeaderBuilder struct {
	headers map[string]string
}

// NewHeaderBuilder creates a fresh HeaderBuilder.
func NewHeaderBuilder() *HeaderBuilder {
	return &HeaderBuilder{headers: make(map[string]string)}
}

// Set adds or overwrites a header.
func (h *HeaderBuilder) Set(name, value string) *HeaderBuilder {
	h.headers[name] = value
	return h
}

// Build returns the final header map.
func (h *HeaderBuilder) Build() map[string]string {
	out := make(map[string]string, len(h.headers))
	for k, v := range h.headers {
		out[k] = v
	}
	return out
}
