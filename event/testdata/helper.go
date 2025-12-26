// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package testdata. helper provides test helpers.
package testdata

import "github.com/seyallius/gossip/event"

// -------------------------------------------- Event Data Structs --------------------------------------------

// UserCreatedData contains information about a newly created user.
type UserCreatedData struct {
	UserID       string
	Username     string
	Email        string
	Organization string
}

// LoginSuccessData contains information about a successful login.
type LoginSuccessData struct {
	UserID       string
	Username     string
	Organization string
	IPAddress    string
	UserAgent    string
}

// LoginFailedData contains information about a failed login attempt.
type LoginFailedData struct {
	Username  string
	IPAddress string
	Reason    string
}

// TokenRevokedData contains information about a revoked token.
type TokenRevokedData struct {
	TokenID   string
	UserID    string
	RevokedBy string
	Reason    string
}

// UserUpdatedData contains information about user profile updates.
type UserUpdatedData struct {
	UserID        string
	Username      string
	ChangedFields []string
	UpdatedBy     string
}

// Event type constants for different domains in the system.
// This is intended for example you can extend it as you want.
// Simply wrap your string in EventType.
//
//goland:noinspection GoCommentStart
const (
	// Authentication Events
	AuthEventUserCreated     event.EventType = "auth.user.created"
	AuthEventLoginSuccess    event.EventType = "auth.login.success"
	AuthEventLoginFailed     event.EventType = "auth.login.failed"
	AuthEventLogout          event.EventType = "auth.logout"
	AuthEventPasswordChanged event.EventType = "auth.password.changed"
	AuthEventTokenRevoked    event.EventType = "auth.token.revoked"

	// User Management Events
	UserEventUpdated  event.EventType = "user.updated"
	UserEventDeleted  event.EventType = "user.deleted"
	UserEventEnabled  event.EventType = "user.enabled"
	UserEventDisabled event.EventType = "user.disabled"

	// Organization Events
	OrgEventCreated event.EventType = "org.created"
	OrgEventUpdated event.EventType = "org.updated"
	OrgEventDeleted event.EventType = "org.deleted"

	// Permission Events
	PermissionEventGranted event.EventType = "permission.granted"
	PermissionEventRevoked event.EventType = "permission.revoked"
)
