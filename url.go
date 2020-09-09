// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"encoding/json"
	"fmt"
	gourl "net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/juju/errors"
	"gopkg.in/juju/names.v3"
	"gopkg.in/mgo.v2/bson"
)

// Schema represents the different types of valid schemas.
type Schema string

const (
	// CharmStore schema represents the original schema for a charm URL
	CharmStore Schema = "cs"

	// Local represents a local charm URL, describes as a file system path.
	Local Schema = "local"

	// CharmHub schema represents the new schema for another unique charm store.
	CharmHub Schema = "ch"

	// HTTP refers to the HTTP schema that is used for the V2 of the charm URL.
	HTTP Schema = "http"

	// HTTPS refers to the HTTP schema that is used for the V2 of the charm URL.
	HTTPS Schema = "https"
)

// Prefix creates a url with the given prefix, useful for typed schemas.
func (s Schema) Prefix(url string) string {
	return fmt.Sprintf("%s:%s", s, url)
}

// Matches attempts to compare if a schema string matches the schema.
func (s Schema) Matches(other string) bool {
	return string(s) == other
}

func (s Schema) String() string {
	return string(s)
}

var (
	// DefaultSchema for the charm package.
	// It's used as the fallback for the absence of a schema in a URL.
	DefaultSchema = CharmHub
)

// Location represents a charm location, which must declare a path component
// and a string representation.
type Location interface {
	Path() string
	String() string
}

// URL represents a charm or bundle location:
//
//     cs:~joe/oneiric/wordpress
//     cs:oneiric/wordpress-42
//     local:oneiric/wordpress
//     cs:~joe/wordpress
//     cs:wordpress
//     cs:precise/wordpress-20
//     cs:development/precise/wordpress-20
//     cs:~joe/development/wordpress
//     ch:wordpress
//
type URL struct {
	Schema   string // "cs", "ch" or "local".
	User     string // "joe".
	Name     string // "wordpress".
	Revision int    // -1 if unset, N otherwise.
	Series   string // "precise" or "" if unset; "bundle" if it's a bundle.
}

var (
	validSeries = regexp.MustCompile("^[a-z]+([a-z0-9]+)?$")
	validName   = regexp.MustCompile("^[a-z][a-z0-9]*(-[a-z0-9]*[a-z][a-z0-9]*)*$")
)

// ValidateSchema returns an error if the schema is invalid.
//
// Valid schemas for the URL are:
// - cs: charm store
// - ch: charm hub
// - local: local file
//
// http and https are not valid schemas, as they compiled to V1 charm URLs.
func ValidateSchema(schema string) error {
	switch schema {
	// ignore http/https schemas.
	case CharmStore.String(), CharmHub.String(), Local.String():
		return nil
	}
	return errors.NotValidf("schema %q", schema)
}

// IsValidSeries reports whether series is a valid series in charm or bundle
// URLs.
func IsValidSeries(series string) bool {
	return validSeries.MatchString(series)
}

// ValidateSeries returns an error if the given series is invalid.
func ValidateSeries(series string) error {
	if IsValidSeries(series) == false {
		return errors.NotValidf("series name %q", series)
	}
	return nil
}

// IsValidName reports whether name is a valid charm or bundle name.
func IsValidName(name string) bool {
	return validName.MatchString(name)
}

// ValidateName returns an error if the given name is invalid.
func ValidateName(name string) error {
	if IsValidName(name) == false {
		return errors.NotValidf("name %q", name)
	}
	return nil
}

// WithRevision returns a URL equivalent to url but with Revision set
// to revision.
func (u *URL) WithRevision(revision int) *URL {
	urlCopy := *u
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
//
// Additionally, fully-qualified charmstore URLs are supported; note that this
// currently assumes that they will map to jujucharms.com (that is,
// fully-qualified URLs currently map to the 'cs' schema):
//
//    https://jujucharms.com/name
//    https://jujucharms.com/name/series
//    https://jujucharms.com/name/revision
//    https://jujucharms.com/name/series/revision
//    https://jujucharms.com/u/user/name
//    https://jujucharms.com/u/user/name/series
//    https://jujucharms.com/u/user/name/revision
//    https://jujucharms.com/u/user/name/series/revision
//
// A missing schema is assumed to be 'cs'.
func ParseURL(url string) (*URL, error) {
	// Check if we're dealing with a v1 or v2 URL.
	u, err := gourl.Parse(url)
	if err != nil {
		return nil, errors.Errorf("cannot parse charm or bundle URL: %q", url)
	}
	if u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return nil, errors.Errorf("charm or bundle URL %q has unrecognized parts", url)
	}
	var curl *URL
	switch {
	case CharmHub.Matches(u.Scheme):
		// Handle talking to the new style of the schema.
		curl, err = parseIdentifierURL(u)
	case u.Opaque != "":
		u.Path = u.Opaque
		curl, err = parseV1URL(u, url)
	case CharmStore.Matches(u.Scheme):
		curl, err = parseV1URL(u, url)
	case HTTP.Matches(u.Scheme) || HTTPS.Matches(u.Scheme):
		curl, err = parseHTTPURL(u)
	default:
		// Handle the fact that anything without a prefix is now a CharmHub
		// charm URL.
		curl, err = parseIdentifierURL(u)
	}
	if err != nil {
		return nil, errors.Trace(err)
	}
	if curl.Schema == "" {
		return nil, errors.Errorf("expected schema for charm or bundle URL: %q", url)
	}
	return curl, nil
}

func parseV1URL(url *gourl.URL, originalURL string) (*URL, error) {
	r := URL{
		Schema: CharmStore.String(),
	}
	if url.Scheme != "" {
		r.Schema = url.Scheme
	}
	if err := ValidateSchema(r.Schema); err != nil {
		return nil, errors.Annotatef(err, "cannot parse URL %q", url)
	}

	parts := strings.Split(url.Path[0:], "/")
	if len(parts) < 1 || len(parts) > 4 {
		return nil, errors.Errorf("charm or bundle URL has invalid form: %q", originalURL)
	}

	// ~<username>
	if strings.HasPrefix(parts[0], "~") {
		if Local.Matches(r.Schema) {
			return nil, errors.Errorf("local charm or bundle URL with user name: %q", originalURL)
		}
		r.User, parts = parts[0][1:], parts[1:]
	}

	if len(parts) > 2 {
		return nil, errors.Errorf("charm or bundle URL has invalid form: %q", originalURL)
	}

	// <series>
	if len(parts) == 2 {
		r.Series, parts = parts[0], parts[1:]
		if err := ValidateSeries(r.Series); err != nil {
			return nil, errors.Annotatef(err, "cannot parse URL %q", originalURL)
		}
	}
	if len(parts) < 1 {
		return nil, errors.Errorf("URL without charm or bundle name: %q", originalURL)
	}

	// <name>[-<revision>]
	r.Name, r.Revision = extractRevision(parts[0])
	if r.User != "" && !names.IsValidUser(r.User) {
		return nil, errors.Errorf("charm or bundle URL has invalid user name: %q", originalURL)
	}
	if err := ValidateName(r.Name); err != nil {
		return nil, errors.Annotatef(err, "cannot parse URL %q", url)
	}
	return &r, nil
}

func (u *URL) path() string {
	var parts []string
	if u.User != "" {
		parts = append(parts, fmt.Sprintf("~%s", u.User))
	}
	if u.Series != "" {
		parts = append(parts, u.Series)
	}
	if u.Revision >= 0 {
		parts = append(parts, fmt.Sprintf("%s-%d", u.Name, u.Revision))
	} else {
		parts = append(parts, u.Name)
	}
	return strings.Join(parts, "/")
}

// FullPath returns the full path of a URL path including the schema.
func (u *URL) FullPath() string {
	return fmt.Sprintf("%s:%s", u.Schema, u.Path())
}

// Path returns the path of the URL without the schema.
func (u *URL) Path() string {
	return u.path()
}

// String returns the string representation of the URL.
// To keep backwards compatibility with older schema versions (CharmStore), we
// output the FullPath of the URL.
// For new CharmHub integrations, we only want the Path.
// That way we can hide the schema from the user.
func (u *URL) String() string {
	if CharmHub.Matches(u.Schema) {
		return u.Path()
	}
	return u.FullPath()
}

// GetBSON turns u into a bson.Getter so it can be saved directly
// on a MongoDB database with mgo.
//
// TODO (stickupkid): This should not be here, as this is purely for mongo
// data stores and that should be implemented at the site of data store, not
// dependant on the library.
func (u *URL) GetBSON() (interface{}, error) {
	if u == nil {
		return nil, nil
	}
	return u.String(), nil
}

// SetBSON turns u into a bson.Setter so it can be loaded directly
// from a MongoDB database with mgo.
//
// TODO (stickupkid): This should not be here, as this is purely for mongo
// data stores and that should be implemented at the site of data store, not
// dependant on the library.
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

// MarshalJSON will marshal the URL into a slice of bytes in a JSON
// representation.
func (u *URL) MarshalJSON() ([]byte, error) {
	if u == nil {
		panic("cannot marshal nil *charm.URL")
	}
	return json.Marshal(u.FullPath())
}

// UnmarshalJSON will unmarshal the URL from a JSON representation.
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

// MarshalText implements encoding.TextMarshaler by
// returning u.FullPath()
func (u *URL) MarshalText() ([]byte, error) {
	if u == nil {
		return nil, nil
	}
	return []byte(u.FullPath()), nil
}

// UnmarshalText implements encoding.TestUnmarshaler by
// parsing the data with ParseURL.
func (u *URL) UnmarshalText(data []byte) error {
	url, err := ParseURL(string(data))
	if err != nil {
		return err
	}
	*u = *url
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

// RewriteURL turns a HTTP(s) URL into a charm URL.
//
// Fully-qualified charmstore URLs are supported; note that this
// currently assumes that they will map to jujucharms.com (that is,
// fully-qualified URLs currently map to the 'cs' schema):
//
//    https://jujucharms.com/name -> cs:name
//    https://jujucharms.com/name/series -> cs:series/name
//    https://jujucharms.com/name/revision -> cs:name-revision
//    https://jujucharms.com/name/series/revision -> cs:series/name-revision
//    https://jujucharms.com/u/user/name -> cs:~user/name
//    https://jujucharms.com/u/user/name/series -> cs:~user/series/name
//    https://jujucharms.com/u/user/name/revision -> cs:~user/name-revision
//    https://jujucharms.com/u/user/name/series/revision -> cs:~user/series/name-revision
//
// A missing schema is assumed to be 'cs'.
func RewriteURL(url string) (string, error) {
	u, err := gourl.Parse(url)
	if err != nil {
		return "", errors.Errorf("cannot parse charm or bundle URL: %q", url)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.Errorf("unexpected url schema %q", u.Scheme)
	}
	if u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return "", errors.Errorf("charm or bundle URL %q has unrecognized parts", url)
	}
	httpURL, err := parseHTTPURL(u)
	if err != nil {
		return "", errors.Trace(err)
	}
	return httpURL.String(), nil
}

func parseHTTPURL(url *gourl.URL) (*URL, error) {
	r := URL{
		Schema: CharmStore.String(),
	}

	parts := strings.Split(strings.Trim(url.Path, "/"), "/")
	if parts[0] == "u" {
		if len(parts) < 3 {
			return nil, errors.Errorf(`charm or bundle URL %q malformed, expected "/u/<user>/<name>"`, url)
		}
		r.User, parts = parts[1], parts[2:]
	}

	r.Name, parts = parts[0], parts[1:]
	r.Revision = -1
	if len(parts) > 0 {
		revision, err := strconv.Atoi(parts[0])
		if err == nil {
			r.Revision = revision
		} else {
			r.Series, parts = parts[0], parts[1:]
			if err := ValidateSeries(r.Series); err != nil {
				return nil, errors.Annotatef(err, "cannot parse URL %q", url)
			}
			if len(parts) == 1 {
				r.Revision, err = strconv.Atoi(parts[0])
				if err != nil {
					return nil, errors.Errorf("charm or bundle URL has malformed revision: %q in %q", parts[0], url)
				}
			} else if len(parts) != 0 {
				return nil, errors.Errorf("charm or bundle URL has invalid form: %q", url)
			}
		}
	}

	if r.User != "" && !names.IsValidUser(r.User) {
		return nil, errors.Errorf("charm or bundle URL has invalid user name: %q", url)
	}
	if err := ValidateName(r.Name); err != nil {
		return nil, errors.Annotatef(err, "cannot parse URL %q", url)
	}
	return &r, nil
}

func parseIdentifierURL(url *gourl.URL) (*URL, error) {
	r := URL{
		Schema:   CharmHub.String(),
		Revision: -1,
	}

	path := url.Path
	if url.Opaque != "" {
		path = url.Opaque
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 1 {
		return nil, errors.Errorf(`charm or bundle URL %q malformed, expected "<name>"`, url)
	}

	r.Name, r.Revision = extractRevision(parts[0])
	if err := ValidateName(r.Name); err != nil {
		return nil, errors.Annotatef(err, "cannot parse URL %q", url)
	}
	return &r, nil
}

// EnsureSchema will ensure that the scheme for a given URL is correct and
// valid.
func EnsureSchema(url string) (string, error) {
	u, err := gourl.Parse(url)
	if err != nil {
		return "", errors.Errorf("cannot parse charm or bundle URL: %q", url)
	}
	switch Schema(u.Scheme) {
	case CharmStore, CharmHub, Local, HTTP, HTTPS:
		return url, nil
	case Schema(""):
		// If the schema is empty, we fall back to the default schema.
		return DefaultSchema.Prefix(url), nil
	default:
		return "", errors.NotValidf("schema %q", u.Scheme)
	}
}

func extractRevision(name string) (string, int) {
	revision := -1
	for i := len(name) - 1; i > 0; i-- {
		c := name[i]
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '-' && i != len(name)-1 {
			var err error
			revision, err = strconv.Atoi(name[i+1:])
			if err != nil {
				panic(err) // We just checked it was right.
			}
			name = name[:i]
		}
		break
	}
	return name, revision
}
