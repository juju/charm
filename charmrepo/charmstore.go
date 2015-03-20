// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmrepo

import (
	"crypto/sha512"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/juju/utils"
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

// Get implements Interface.Get.
func (s *CharmStore) Get(curl *charm.URL) (charm.Charm, error) {
	// Disable bundle deployment.
	if curl.Series == "bundle" {
		return nil, errgo.Newf("expected a charm URL, got bundle URL %q", curl)
	}

	// Prepare the cache directory and retrieve the charm.
	if err := os.MkdirAll(s.cacheDir, os.FileMode(0755)); err != nil {
		return nil, errgo.Notef(err, "cannot create the cache directory")
	}
	r, id, expectHash, _, err := s.client.GetArchive(curl.Reference())
	if err != nil {
		return nil, errgo.Notef(err, "cannot retrieve charm")
	}
	defer r.Close()

	// Check if the archive already exists in the cache.
	path := filepath.Join(s.cacheDir, charm.Quote(id.String())+".charm")
	if verifyHash384(path, expectHash) == nil {
		return charm.ReadCharmArchive(path)
	}

	// Verify and save the new archive.
	f, err := ioutil.TempFile(s.cacheDir, "charm-download")
	if err != nil {
		return nil, errgo.Notef(err, "cannot make temporary file")
	}
	defer f.Close()
	hash := sha512.New384()
	_, err = io.Copy(io.MultiWriter(hash, f), r)
	if err != nil {
		return nil, errgo.Notef(err, "cannot read charm archive")
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != expectHash {
		return nil, errgo.Newf("hash mismatch; network corruption?")
	}

	// Move the archive to the expected place, and return the charm.
	if err := utils.ReplaceFile(f.Name(), path); err != nil {
		return nil, errgo.Notef(err, "cannot move the charm archive")
	}
	return charm.ReadCharmArchive(path)
}

func verifyHash384(path, expectHash string) error {
	f, err := os.Open(path)
	if err != nil {
		return errgo.Mask(err)
	}
	defer f.Close()
	hash := sha512.New384()
	if _, err := io.Copy(hash, f); err != nil {
		return errgo.Mask(err)
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != expectHash {
		return errgo.Newf("hash mismatch for %q", path)
	}
	return nil
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

// WithTestMode returns a repository Interface where test mode is enabled,
// meaning charm store download stats are not increased when charms are
// retrieved.
func (s *CharmStore) WithTestMode() Interface {
	newRepo := *s
	newRepo.client.DisableStats()
	return &newRepo
}
