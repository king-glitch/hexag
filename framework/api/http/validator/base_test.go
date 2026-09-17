package validator

import (
	"testing"

	serviceerrors "github.com/king-glitch/hexag/framework/api/service/errors"
	hexports "github.com/king-glitch/hexag/framework/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStatus string

const (
	TestStatusActive  TestStatus = "active"
	TestStatusPending TestStatus = "pending"
)

func (s TestStatus) IsValid() bool {
	switch s {
	case TestStatusActive, TestStatusPending:
		return true
	default:
		return false
	}
}

var _ hexports.Validatable = TestStatusActive

type SampleRequest struct {
	Name   string     `json:"name" validate:"required"`
	Status TestStatus `json:"status"` // Note: No validate tag, autodetected!
}

type NestedRequest struct {
	Sample SampleRequest `json:"sample"`
}

func TestValidator_AutodetectsEnum(t *testing.T) {
	v := NewStructValidator()

	// 1. Valid enum passes
	validReq := SampleRequest{
		Name:   "Alice",
		Status: TestStatusActive,
	}
	err := v.Validate(&validReq)
	assert.NoError(t, err)

	// 2. Invalid enum fails automatically without validate tag
	invalidReq := SampleRequest{
		Name:   "Bob",
		Status: TestStatus("banned"),
	}
	err = v.Validate(&invalidReq)
	require.Error(t, err)

	serr, ok := err.(*serviceerrors.ServiceError)
	require.True(t, ok)
	assert.Equal(t, serviceerrors.ServiceErrorCodeValidation, serr.Code)
	assert.Contains(t, serr.Violations, "status")
	assert.Equal(t, "this field contains an invalid value", serr.Violations["status"].Message)

	// 3. Nested struct enum autodetection
	nestedReq := NestedRequest{
		Sample: SampleRequest{
			Name:   "Charlie",
			Status: TestStatus("invalid_val"),
		},
	}
	err = v.Validate(&nestedReq)
	require.Error(t, err)
	serr, ok = err.(*serviceerrors.ServiceError)
	require.True(t, ok)
	assert.Contains(t, serr.Violations, "sample.status")

	// 4. Pointer to enum autodetection
	type PointerRequest struct {
		Status *TestStatus `json:"status"`
	}
	invalidStatus := TestStatus("bad_pointer")
	ptrReq := PointerRequest{Status: &invalidStatus}
	err = v.Validate(&ptrReq)
	require.Error(t, err)
	serr, ok = err.(*serviceerrors.ServiceError)
	require.True(t, ok)
	assert.Contains(t, serr.Violations, "status")

	// 5. Slice of enums autodetection
	type SliceRequest struct {
		Statuses []TestStatus `json:"statuses"`
	}
	sliceReq := SliceRequest{Statuses: []TestStatus{TestStatusActive, TestStatus("bad_slice_item")}}
	err = v.Validate(&sliceReq)
	require.Error(t, err)
	serr, ok = err.(*serviceerrors.ServiceError)
	require.True(t, ok)
	assert.Contains(t, serr.Violations, "statuses[1]")
}
