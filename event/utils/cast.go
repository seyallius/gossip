// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package utils. cast.go - Provides utilities for safely casting event
// data with configurable nil-checking strategies.
package utils

import (
	"fmt"
	"reflect"

	"github.com/seyallius/gossip/v2/event"
	"github.com/seyallius/gossip/v2/event/errors"
)

// ----------------------------------------------- Types -------------------------------------------- //

// NilChecker is a function that returns true if the given value is considered nil.
// This is used to detect "typed nils" (e.g., a nil pointer wrapped in an interface).
type NilChecker func(v any) bool

// CastOption configures the behavior of CastEventData.
type CastOption func(*castConfig)

type castConfig struct {
	nilChecker NilChecker
}

// -------------------------------------------- Public API ------------------------------------------ //

// CastEventData safely casts the event's Data field to type T.
// It returns an error if the type assertion fails or if the data is nil.
//
// By default, it only checks for untyped nils (fast path).
// To check for "typed nils" (which can cause panics later), use CastOptions:
//   - WithReflectNilCheck(): Safest, but uses reflection (slower).
//   - WithCustomNilCheck(): Fastest, provide your own type-assertion based checker.
//   - WithNoNilCheck(): Disables typed nil checking entirely (default).
func CastEventData[T any](e *event.Event, opts ...CastOption) (T, error) {
	var zero T

	// 1. Fast path: Check for untyped nil immediately
	if e.Data == nil {
		return zero, errors.ErrNilData
	}

	// 2. Type assertion
	data, ok := e.Data.(T)
	if !ok {
		return zero, errors.NewTypeAssertionError(
			fmt.Sprintf("%T", zero),
			e.Data,
			fmt.Sprintf("expected %T, got %T for event %s", zero, e.Data, e.Type),
		)
	}

	// 3. Apply functional options
	cfg := &castConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// 4. Typed nil check (if configured)
	if cfg.nilChecker != nil && cfg.nilChecker(data) {
		return zero, errors.ErrNilData
	}

	return data, nil
}

// WithReflectNilCheck uses reflection to check for typed nils.
// This is the safest option but incurs a performance penalty.
// Use this in non-hot paths or when you cannot guarantee publisher behavior.
func WithReflectNilCheck() CastOption {
	return func(c *castConfig) {
		c.nilChecker = ReflectNilChecker
	}
}

// WithNoNilCheck disables typed nil checking entirely.
// This is the fastest option and should be used in hot paths where
// you can guarantee that publishers will never send typed nils.
func WithNoNilCheck() CastOption {
	return func(c *castConfig) {
		c.nilChecker = nil
	}
}

// WithCustomNilCheck allows you to provide your own high-performance nil checker.
// This is ideal for hot paths where you know the exact type and can use a type assertion.
func WithCustomNilCheck(checker NilChecker) CastOption {
	return func(c *castConfig) {
		c.nilChecker = checker
	}
}

// ReflectNilChecker uses reflection to check if a value is a typed nil.
// It safely handles pointers, maps, slices, channels, functions, and interfaces.
func ReflectNilChecker(v any) bool {
	if v == nil {
		return true
	}
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return val.IsNil()
	}
	return false
}
