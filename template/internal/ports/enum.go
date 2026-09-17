package ports

// ExampleStatus demonstrates a domain enum implementing hexports.Validatable.
type ExampleStatus string

const (
	ExampleStatusActive   ExampleStatus = "active"
	ExampleStatusInactive ExampleStatus = "inactive"
)

func (s ExampleStatus) IsValid() bool {
	switch s {
	case ExampleStatusActive, ExampleStatusInactive:
		return true
	default:
		return false
	}
}
