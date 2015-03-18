// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmrepo

import (
	"net/http"
	"net/url"

	"gopkg.in/errgo.v1"
	"gopkg.in/juju/charmstore.v4/csclient"
	"gopkg.in/juju/charmstore.v4/params"

	"gopkg.in/juju/charm.v5-unstable"
)

// CharmStore is a repository Interface that provides access to the public Juju
// charm store.
type CharmStore struct {
	client   *csclient.Client
	cacheDir string
	testMode bool
}

var _ Interface = (*CharmStore)(nil)

// NewCharmStoreParams holds parameters for instantiating a new CharmStore.
type NewCharmStoreParams struct {
	// URL holds the root endpoint URL of the charm store,
	// with no trailing slash, not including the version.
	// For example https://api.jujucharms.com/charmstore
	// If empty, the default charm store client location is used.
	URL string

	// HTTPClient holds the HTTP client to use when making
	// requests to the store. If nil, httpbakery.NewHTTPClient will
	// be used.
	HTTPClient *http.Client

	// VisitWebPage is called when authorization requires that
	// the user visits a web page to authenticate themselves.
	// If nil, a default function that returns an error will be used.
	VisitWebPage func(url *url.URL) error

	// CacheDir holds the charm cache directory path where to store retrieved
	// charms.
	CacheDir string
}

// NewCharmStore creates and returns a charm store repository.
// The given parameters are used to instantiate the charm store.
func NewCharmStore(p NewCharmStoreParams) (Interface, error) {
	if p.CacheDir == "" {
		return nil, errgo.New("charm cache directory path is empty")
	}
	return &CharmStore{
		client: csclient.New(csclient.Params{
			URL:          p.URL,
			HTTPClient:   p.HTTPClient,
			VisitWebPage: p.VisitWebPage,
		}),
		cacheDir: p.CacheDir,
	}, nil
}

var notImplemented = errgo.New("not implemented")

// Get implements Interface.Get.
func (s *CharmStore) Get(curl *charm.URL) (charm.Charm, error) {
	return nil, notImplemented
}

// Latest implements Interface.Latest.
func (s *CharmStore) Latest(curls ...*charm.URL) ([]CharmRevision, error) {
	if len(curls) == 0 {
		return nil, nil
	}

	// Prepare the request to the charm store.
	urls := make([]string, len(curls))
	values := url.Values{}
	values.Add("include", "id-revision")
	values.Add("include", "hash256")
	for i, curl := range curls {
		url := curl.WithRevision(-1).String()
		urls[i] = url
		values.Add("id", url)
	}
	u := url.URL{
		Path:     "/meta/any",
		RawQuery: values.Encode(),
	}

	// Execute the request and retrieve results.
	var results map[string]struct {
		Meta struct {
			IdRevision params.IdRevisionResponse `json:"id-revision"`
			Hash256    params.HashResponse       `json:"hash256"`
		}
	}
	if err := s.client.Get(u.String(), &results); err != nil {
		return nil, errgo.Notef(err, "cannot get metadata from the charm store")
	}

	// Build the response.
	responses := make([]CharmRevision, len(curls))
	for i, url := range urls {
		result, found := results[url]
		if !found {
			responses[i] = CharmRevision{
				Err: CharmNotFound(url),
			}
			continue
		}
		responses[i] = CharmRevision{
			Revision: result.Meta.IdRevision.Revision,
			Sha256:   result.Meta.Hash256.Sum,
		}
	}
	return responses, nil
}

// Resolve implements Interface.Resolve.
func (s *CharmStore) Resolve(ref *charm.Reference) (*charm.URL, error) {
	var result struct {
		IdSeries params.IdSeriesResponse
	}
	if _, err := s.client.Meta(ref, &result); err != nil {
		return nil, errgo.Notef(err, "cannot get metadata from the charm store")
	}
	url := charm.URL(*ref)
	url.Series = result.IdSeries.Series
	return &url, nil
}

// URL returns the root endpoint URL of the charm store.
func (s *CharmStore) URL() string {
	return s.client.ServerURL()
}

// WithTestMode returns a repository Interface where testMode is set to value
// passed to this method.
func (s *CharmStore) WithTestMode(testMode bool) Interface {
	newRepo := *s
	newRepo.testMode = testMode
	return &newRepo
}
