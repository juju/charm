// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource_test

import (
	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v6-unstable/resource"
)

var _ = gc.Suite(&FingerprintSuite{})

type FingerprintSuite struct{}

func (s *FingerprintSuite) TestNewFingerprint(c *gc.C) {
	expected := []byte("123456789012345678901234567890123456789012345678")
	fp := resource.NewFingerprint(expected)
	raw := fp.Raw()

	c.Check(raw, jc.DeepEquals, expected)
}

func (s *FingerprintSuite) TestBuildFingerprint(c *gc.C) {
	fp, err := resource.BuildFingerprint([]byte("spamspamspam"))
	c.Assert(err, jc.ErrorIsNil)
	raw := fp.Raw()

	c.Logf("%q", raw)
	c.Check(raw, jc.DeepEquals, []byte("\xfa\xc0\x9f}gѽ0\xf4\x115\xc1o\xc12\xe5\xb65\x98\x8c\x16/57\x86\xca(\xec\x06\x05\xd4\xd8\xedUy\x9d]\x02\xe9'TsWK\xf3uIu"))
}

func (s *FingerprintSuite) TestHex(c *gc.C) {
	//raw := []byte("123456789012345678901234567890123456789012345678")
	raw := []byte("abcdefghijklmnopqrstuvwxyz1234567890123456789012")
	fp := resource.NewFingerprint(raw)
	hex := fp.Hex()

	c.Check(hex, jc.DeepEquals, "6162636465666768696a6b6c6d6e6f707172737475767778797a31323334353637383930313233343536373839303132")
}

func (s *FingerprintSuite) TestValidateOkay(c *gc.C) {
	fp := resource.NewFingerprint([]byte("123456789012345678901234567890123456789012345678"))
	err := fp.Validate()

	c.Check(err, jc.ErrorIsNil)
}

func (s *FingerprintSuite) TestValidateZero(c *gc.C) {
	var fp resource.Fingerprint
	err := fp.Validate()

	c.Check(err, jc.Satisfies, errors.IsNotValid)
}

func (s *FingerprintSuite) TestValidateTooSmall(c *gc.C) {
	fp := resource.NewFingerprint([]byte("12345678901234567890123456789012345678901234567"))
	err := fp.Validate()

	c.Check(err, jc.Satisfies, errors.IsNotValid)
	c.Check(err, gc.ErrorMatches, `.*too small.*`)
}

func (s *FingerprintSuite) TestValidateTooBig(c *gc.C) {
	fp := resource.NewFingerprint([]byte("1234567890123456789012345678901234567890123456789"))
	err := fp.Validate()

	c.Check(err, jc.Satisfies, errors.IsNotValid)
	c.Check(err, gc.ErrorMatches, `.*too big.*`)
}
