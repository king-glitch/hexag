package errors

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceError_ChainPreservation(t *testing.T) {
	sentinel := errors.New("archive item not found")
	RegisterSentinel(sentinel, ServiceErrorCodeNotFound)

	// Step 1: innermost repository / rule returns sentinel error, service wraps with NewServiceErrorFromCause
	serr1 := NewServiceErrorFromCause(sentinel, "failed to get archive item")
	assert.Equal(t, "archive item not found", serr1.Message)
	assert.Equal(t, ServiceErrorCodeNotFound, serr1.Code)
	assert.Equal(t, "failed to get archive item: archive item not found", serr1.Stack)
	assert.Equal(t, "failed to get archive item: archive item not found", serr1.Error())

	// Step 2: caller wraps with serr.Wrap (e.g. expedition service wrapping storage service reserve failure)
	wrappedSerr := serr1.Wrap("failed to reserve expedition rewards")
	assert.Equal(t, "archive item not found", wrappedSerr.Message)
	assert.Equal(t, ServiceErrorCodeNotFound, wrappedSerr.Code)
	assert.Equal(t, "failed to reserve expedition rewards: failed to get archive item: archive item not found", wrappedSerr.Stack)
	assert.Equal(t, "failed to reserve expedition rewards: failed to get archive item: archive item not found", wrappedSerr.Error())

	// Step 3: mongo transaction runner wraps the error returned by fn()
	// in runner: return errors.Wrap(err, "failed to run transaction")
	txErr := errors.Wrap(wrappedSerr, "failed to run transaction")
	assert.Equal(t, "failed to run transaction: failed to reserve expedition rewards: failed to get archive item: archive item not found", txErr.Error())

	// Step 4: expedition service catches transaction error and calls NewServiceErrorFromCause
	outerSerr := NewServiceErrorFromCause(txErr, "failed to end expedition entry")
	assert.Equal(t, "archive item not found", outerSerr.Message)
	assert.Equal(t, ServiceErrorCodeNotFound, outerSerr.Code)
	assert.Equal(t, "failed to end expedition entry: failed to run transaction: failed to reserve expedition rewards: failed to get archive item: archive item not found", outerSerr.Stack)
	assert.Equal(t, "failed to end expedition entry: failed to run transaction: failed to reserve expedition rewards: failed to get archive item: archive item not found", outerSerr.Error())

	// Step 5: route handler wraps error
	finalSerr := outerSerr.Wrap("failed to end expedition entry")
	assert.Equal(t, "archive item not found", finalSerr.Message)
	assert.Equal(t, ServiceErrorCodeNotFound, finalSerr.Code)
	expectedStack := "failed to end expedition entry: failed to end expedition entry: failed to run transaction: failed to reserve expedition rewards: failed to get archive item: archive item not found"
	assert.Equal(t, expectedStack, finalSerr.Stack)
	assert.Equal(t, expectedStack, finalSerr.Error())

	// Unwrapping / Cause / Is checks
	assert.True(t, errors.Is(finalSerr, sentinel))
	assert.Equal(t, sentinel, errors.Cause(finalSerr))
}

func TestServiceError_PreservesCodeAndViolationsWhenWrapped(t *testing.T) {
	serr := NewServiceError(ServiceErrorCodeValidation, errors.New("invalid payload"))
	serr.WithStatusCode(422)
	serr.AddError("field1", "cannot be blank", errors.New("blank"))

	wrapped := NewServiceErrorFromCause(serr, "validation failed")
	assert.Equal(t, ServiceErrorCodeValidation, wrapped.Code)
	assert.Equal(t, 422, wrapped.StatusCode)
	require.Contains(t, wrapped.Violations, "field1")
	assert.Equal(t, "cannot be blank", wrapped.Violations["field1"].Message)
}

func TestServiceError_NilAndEmptyCases(t *testing.T) {
	var nilSerr *ServiceError
	assert.Nil(t, nilSerr.Wrap("msg"))
	assert.Nil(t, nilSerr.Unwrap())
	assert.Nil(t, nilSerr.Cause())
	assert.Equal(t, "", nilSerr.Error())

	emptySerr := NewServiceError(ServiceErrorCodeInternal, nil)
	assert.Equal(t, "failed to process the request", emptySerr.Message)
	assert.Equal(t, "failed to process the request", emptySerr.Error())

	wrappedEmpty := emptySerr.Wrap("extra context")
	assert.Equal(t, "extra context: failed to process the request", wrappedEmpty.Stack)
	assert.Equal(t, "extra context: failed to process the request", wrappedEmpty.Error())
}
