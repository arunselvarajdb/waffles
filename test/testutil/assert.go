package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertHelper provides common assertion helpers
type AssertHelper struct {
	t       *testing.T
	assert  *assert.Assertions
	require *require.Assertions
}

// NewAssertHelper creates a new assertion helper
func NewAssertHelper(t *testing.T) *AssertHelper {
	return &AssertHelper{
		t:       t,
		assert:  assert.New(t),
		require: require.New(t),
	}
}

// NoError asserts no error occurred
func (a *AssertHelper) NoError(err error, msgAndArgs ...interface{}) {
	a.require.NoError(err, msgAndArgs...)
}

// Error asserts an error occurred
func (a *AssertHelper) Error(err error, msgAndArgs ...interface{}) {
	a.require.Error(err, msgAndArgs...)
}

// Equal asserts equality
func (a *AssertHelper) Equal(expected, actual interface{}, msgAndArgs ...interface{}) {
	a.assert.Equal(expected, actual, msgAndArgs...)
}

// NotEmpty asserts value is not empty
func (a *AssertHelper) NotEmpty(obj interface{}, msgAndArgs ...interface{}) {
	a.assert.NotEmpty(obj, msgAndArgs...)
}

// Contains asserts string contains substring
func (a *AssertHelper) Contains(s, substr string, msgAndArgs ...interface{}) {
	a.assert.Contains(s, substr, msgAndArgs...)
}
