// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmrepo

import (
	"fmt"

	"gopkg.in/juju/charm.v5-unstable"
)

// InfoResponse is sent by the charm store in response to charm-info requests.
type InfoResponse struct {
	CanonicalURL string   `json:"canonical-url,omitempty"`
	Revision     int      `json:"revision"` // Zero is valid. Can't omitempty.
	Sha256       string   `json:"sha256,omitempty"`
	Digest       string   `json:"digest,omitempty"`
	Errors       []string `json:"errors,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

// EventResponse is sent by the charm store in response to charm-event requests.
type EventResponse struct {
	Kind     string   `json:"kind"`
	Revision int      `json:"revision"` // Zero is valid. Can't omitempty.
	Digest   string   `json:"digest,omitempty"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Time     string   `json:"time,omitempty"`
}

// CharmRevision holds the revision number of a charm and any error
// encountered in retrieving it.
type CharmRevision struct {
	Revision int
	Sha256   string
	Err      error
}

// NotFoundError represents an error indicating that the requested data wasn't found.
type NotFoundError struct {
	msg string
}

func (e *NotFoundError) Error() string {
	return e.msg
}

func repoNotFound(path string) error {
	return &NotFoundError{fmt.Sprintf("no repository found at %q", path)}
}

func charmNotFound(curl *charm.URL, repoPath string) error {
	return &NotFoundError{fmt.Sprintf("charm not found in %q: %s", repoPath, curl)}
}

func CharmNotFound(url string) error {
	return &NotFoundError{
		msg: "charm not found: " + url,
	}
}
