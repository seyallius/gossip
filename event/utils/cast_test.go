// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package utils. cast_test.go provides comprehensive tests for the casting
// utilities and nil-checking strategies.
package utils

import (
	"testing"

	"github.com/seyallius/gossip/v2/event"
	"github.com/seyallius/gossip/v2/event/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----------------------------------------------- Types -------------------------------------------- //

// UserData is a mock struct for testing purposes.
type UserData struct {
	ID   string
	Name string
}

// FastUserDataNilChecker is an example of a zero-allocation, reflection-free nil checker.
func FastUserDataNilChecker(v any) bool {
	u, ok := v.(*UserData)
	if !ok {
		return false
	}
	return u == nil
}

// ---------------------------------------------- Tests -------------------------------------------- //

// TestCastEventData_Success verifies that valid data is cast correctly.
func TestCastEventData_Success(t *testing.T) {
	evt := event.NewEvent("user.created", &UserData{ID: "123", Name: "Alice"})

	data, err := CastEventData[*UserData](evt)
	require.NoError(t, err)
	assert.Equal(t, "123", data.ID)
}

// TestCastEventData_UntypedNil verifies that untyped nils return ErrNilData.
func TestCastEventData_UntypedNil(t *testing.T) {
	evt := event.NewEvent("user.created", nil)

	data, err := CastEventData[*UserData](evt)
	assert.ErrorIs(t, err, errors.ErrNilData)
	assert.Nil(t, data)
}

// TestCastEventData_TypeAssertionFails verifies that wrong types return TypeAssertionError.
func TestCastEventData_TypeAssertionFails(t *testing.T) {
	evt := event.NewEvent("user.created", "not a struct")

	data, err := CastEventData[*UserData](evt)
	assert.Error(t, err)
	assert.True(t, errors.IsTypeAssertion(err))
	assert.Nil(t, data)
}

// TestCastEventData_TypedNil_Default proves that typed nils pass through by default (fast path).
func TestCastEventData_TypedNil_Default(t *testing.T) {
	// A typed nil: the interface holds a type (*UserData) but the value is nil.
	var u *UserData = nil
	evt := event.NewEvent("user.created", u)

	// By default, typed nils pass through because any(data) != nil
	data, err := CastEventData[*UserData](evt)
	require.NoError(t, err)
	assert.Nil(t, data) // It's a typed nil pointer!
}

// TestCastEventData_TypedNil_WithReflect proves that reflection catches typed nils.
func TestCastEventData_TypedNil_WithReflect(t *testing.T) {
	var u *UserData = nil
	evt := event.NewEvent("user.created", u)

	data, err := CastEventData[*UserData](evt, WithReflectNilCheck())
	assert.ErrorIs(t, err, errors.ErrNilData)
	assert.Nil(t, data)
}

// TestCastEventData_TypedNil_WithCustomChecker proves that custom checkers catch typed nils without reflection.
func TestCastEventData_TypedNil_WithCustomChecker(t *testing.T) {
	var u *UserData = nil
	evt := event.NewEvent("user.created", u)

	data, err := CastEventData[*UserData](evt, WithCustomNilCheck(FastUserDataNilChecker))
	assert.ErrorIs(t, err, errors.ErrNilData)
	assert.Nil(t, data)
}

// TestCastEventData_ValueType verifies that value types (non-pointers) work correctly.
func TestCastEventData_ValueType(t *testing.T) {
	// Value types (non-pointers) cannot be nil in Go.
	evt := event.NewEvent("user.created", UserData{ID: "123", Name: "Alice"})

	data, err := CastEventData[UserData](evt)
	require.NoError(t, err)
	assert.Equal(t, "123", data.ID)
}
