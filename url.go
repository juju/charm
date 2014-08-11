// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/juju/names"
	"gopkg.in/mgo.v2/bson"
)

// Location represents a charm location, which must declare a path component
// and a string representaion.
type Location interface {
	Path() string
	String() string
}

// URL represents a fully resolved charm location with a specific series, such
// as:
//
//     cs:~joe/oneiric/wordpress
//     cs:oneiric/wordpress-42
//     local:oneiric/wordpress
//
type URL struct {
	Schema   string // "cs" or "local"
	User     string // "joe"
	Name     string // "wordpress"
	Revision int    // -1 if unset, N otherwise
	Series   string
}

// Reference represents a charm location with a series
// that may be unresolved.
//
//     cs:~joe/wordpress
//     cs:wordpress-42
//     cs:precise/wordpress
type Reference URL

var ErrUnresolvedUrl error = fmt.Errorf("charm url series is not resolved")

var (
	validSeries = regexp.MustCompile("^[a-z]+([a-z0-9]+)?$")
	validName   = regexp.MustCompile("^[a-z][a-z0-9]*(-[a-z0-9]*[a-z][a-z0-9]*)*$")
)

// IsValidSeries returns whether series is a valid series in charm URLs.
func IsValidSeries(series string) bool {
	return validSeries.MatchString(series)
}

// IsValidName returns whether name is a valid charm name.
func IsValidName(name string) bool {
	return validName.MatchString(name)
}

// WithRevision returns a URL equivalent to url but with Revision set
// to revision.
func (url *URL) WithRevision(revision int) *URL {
	urlCopy := *url
	urlCopy.Revision = revision
	return &urlCopy
}

// MustParseURL works like ParseURL, but panics in case of errors.
func MustParseURL(url string) *URL {
	u, err := ParseURL(url)
	if err != nil {
		panic(err)
	}
	return u
}

// ParseURL parses the provided charm URL string into its respective
// structure.
func ParseURL(urlStr string) (*URL, error) {
	r, err := parseReference(urlStr)
	if err != nil {
		return nil, err
	}
	if r.Series == "" {
		return nil, ErrUnresolvedUrl
	}
	if r.Schema == "" {
		return nil, fmt.Errorf("charm URL has no schema: %q", urlStr)
	}
	url, err := r.URL("")
	if err != nil {
		return nil, err // should never happen, because series is set.
	}
	return url, nil
}

// URL returns a full URL from the reference, creating
// a new URL value if necessary with the given default
// series. It returns an error if ref does not specify
// a series and defaultSeries is empty.
func (ref *Reference) URL(defaultSeries string) (*URL, error) {
	if ref.Series != "" {
		return (*URL)(ref), nil
	}
	if defaultSeries == "" {
		return nil, ErrUnresolvedUrl
	}
	if !IsValidSeries(defaultSeries) {
		return nil, fmt.Errorf("default series %q is invalid", defaultSeries)
	}
	url := *(*URL)(ref)
	url.Series = defaultSeries
	return &url, nil
}

// ParseReference returns a charm reference inferred from src. The provided
// src may be a valid URL or it may be an alias in one of the following formats:
//
//    name
//    name-revision
//    series/name
//    series/name-revision
//    schema:name
//    schema:name-revision
//    schema:~user/name
//    schema:~user/name-revision
//
// A missing schema is assumed to be 'cs'.
func ParseReference(url string) (*Reference, error) {
	ref, err := parseReference(url)
	if err != nil {
		return nil, err
	}
	if ref.Schema == "" {
		ref.Schema = "cs"
	}
	return ref, nil
}

func parseReference(url string) (*Reference, error) {
	var r Reference
	i := strings.Index(url, ":")
	if i >= 0 {
		r.Schema = url[:i]
		if r.Schema != "cs" && r.Schema != "local" {
			return nil, fmt.Errorf("charm URL has invalid schema: %q", url)
		}
		i++
	} else {
		i = 0
	}
	parts := strings.Split(url[i:], "/")
	if len(parts) < 1 || len(parts) > 3 {
		return nil, fmt.Errorf("charm URL has invalid form: %q", url)
	}

	// ~<username>
	if strings.HasPrefix(parts[0], "~") {
		if r.Schema == "local" {
			return nil, fmt.Errorf("local charm URL with user name: %q", url)
		}
		r.User = parts[0][1:]
		if !names.IsValidUser(r.User) {
			return nil, fmt.Errorf("charm URL has invalid user name: %q", url)
		}
		parts = parts[1:]
	}
	if len(parts) > 2 {
		return nil, fmt.Errorf("charm URL has invalid form: %q", url)
	}
	// <series>
	if len(parts) == 2 {
		r.Series = parts[0]
		if !IsValidSeries(r.Series) {
			return nil, fmt.Errorf("charm URL has invalid series: %q", url)
		}
		parts = parts[1:]
	}
	if len(parts) < 1 {
		return nil, fmt.Errorf("charm URL without charm name: %q", url)
	}

	// <name>[-<revision>]
	r.Name = parts[0]
	r.Revision = -1
	for i := len(r.Name) - 1; i > 0; i-- {
		c := r.Name[i]
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '-' && i != len(r.Name)-1 {
			var err error
			r.Revision, err = strconv.Atoi(r.Name[i+1:])
			if err != nil {
				panic(err) // We just checked it was right.
			}
			r.Name = r.Name[:i]
		}
		break
	}
	if !IsValidName(r.Name) {
		return nil, fmt.Errorf("charm URL has invalid charm name: %q", url)
	}
	return &r, nil
}

func (r *Reference) path() string {
	var parts []string
	if r.User != "" {
		parts = append(parts, fmt.Sprintf("~%s", r.User))
	}
	if r.Series != "" {
		parts = append(parts, r.Series)
	}
	if r.Revision >= 0 {
		parts = append(parts, fmt.Sprintf("%s-%d", r.Name, r.Revision))
	} else {
		parts = append(parts, r.Name)
	}
	return strings.Join(parts, "/")
}

func (r Reference) Path() string {
	return r.path()
}

// InferURL parses src as a reference, fills out the series in the
// returned URL using defaultSeries if necessary.
//
// This function is deprecated. New code should use ParseReference
// and/or Reference.URL instead.
func InferURL(src, defaultSeries string) (*URL, error) {
	ref, err := ParseReference(src)
	if err != nil {
		return nil, err
	}
	url, err := ref.URL(defaultSeries)
	if err != nil {
		return nil, fmt.Errorf("cannot infer charm URL for %q: %v", src, err)
	}
	return url, nil
}

// Reference returns a reference aliased to u. Note that
// all URLs are valid references.
func (u *URL) Reference() *Reference {
	return (*Reference)(u)
}

func (u *URL) Path() string {
	return (*Reference)(u).path()
}

func (u *URL) String() string {
	return fmt.Sprintf("%s:%s", u.Schema, u.Path())
}

func (r Reference) String() string {
	return fmt.Sprintf("%s:%s", r.Schema, r.Path())
}

// GetBSON turns u into a bson.Getter so it can be saved directly
// on a MongoDB database with mgo.
func (u *URL) GetBSON() (interface{}, error) {
	if u == nil {
		return nil, nil
	}
	return u.String(), nil
}

// SetBSON turns u into a bson.Setter so it can be loaded directly
// from a MongoDB database with mgo.
func (u *URL) SetBSON(raw bson.Raw) error {
	if raw.Kind == 10 {
		return bson.SetZero
	}
	var s string
	err := raw.Unmarshal(&s)
	if err != nil {
		return err
	}
	url, err := ParseURL(s)
	if err != nil {
		return err
	}
	*u = *url
	return nil
}

func (u *URL) MarshalJSON() ([]byte, error) {
	if u == nil {
		panic("cannot marshal nil *charm.URL")
	}
	return json.Marshal(u.String())
}

func (u *URL) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	url, err := ParseURL(s)
	if err != nil {
		return err
	}
	*u = *url
	return nil
}

// GetBSON turns r into a bson.Getter so it can be saved directly
// on a MongoDB database with mgo.
func (r *Reference) GetBSON() (interface{}, error) {
	if r == nil {
		return nil, nil
	}
	return r.String(), nil
}

// SetBSON turns u into a bson.Setter so it can be loaded directly
// from a MongoDB database with mgo.
func (r *Reference) SetBSON(raw bson.Raw) error {
	if raw.Kind == 10 {
		return bson.SetZero
	}
	var s string
	err := raw.Unmarshal(&s)
	if err != nil {
		return err
	}
	ref, err := ParseReference(s)
	if err != nil {
		return err
	}
	*r = *ref
	return nil
}

func (r *Reference) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func (r *Reference) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	ref, err := ParseReference(s)
	if err != nil {
		return err
	}
	*r = *ref
	return nil
}

// Quote translates a charm url string into one which can be safely used
// in a file path.  ASCII letters, ASCII digits, dot and dash stay the
// same; other characters are translated to their hex representation
// surrounded by underscores.
func Quote(unsafe string) string {
	safe := make([]byte, 0, len(unsafe)*4)
	for i := 0; i < len(unsafe); i++ {
		b := unsafe[i]
		switch {
		case b >= 'a' && b <= 'z',
			b >= 'A' && b <= 'Z',
			b >= '0' && b <= '9',
			b == '.',
			b == '-':
			safe = append(safe, b)
		default:
			safe = append(safe, fmt.Sprintf("_%02x_", b)...)
		}
	}
	return string(safe)
}
