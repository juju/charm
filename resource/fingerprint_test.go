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

func (s *FingerprintSuite) TestNewFingerprintOkay(c *gc.C) {
	expected := []byte("123456789012345678901234567890123456789012345678")
	fp, err := resource.NewFingerprint(expected)
	c.Assert(err, jc.ErrorIsNil)
	raw := fp.Bytes()

	c.Check(raw, jc.DeepEquals, expected)
}

func (s *FingerprintSuite) TestNewFingerprintTooSmall(c *gc.C) {
	_, err := resource.NewFingerprint([]byte("12345678901234567890123456789012345678901234567"))

	c.Check(err, jc.Satisfies, errors.IsNotValid)
	c.Check(err, gc.ErrorMatches, `.*too small.*`)
}

func (s *FingerprintSuite) TestNewFingerprintTooBig(c *gc.C) {
	_, err := resource.NewFingerprint([]byte("1234567890123456789012345678901234567890123456789"))

	c.Check(err, jc.Satisfies, errors.IsNotValid)
	c.Check(err, gc.ErrorMatches, `.*too big.*`)
}

func (s *FingerprintSuite) TestGenerateFingerprint(c *gc.C) {
	fp, err := resource.GenerateFingerprint([]byte("spamspamspam"))
	c.Assert(err, jc.ErrorIsNil)
	hex := fp.String()

	c.Check(hex, jc.DeepEquals, "fac09f7d67d1bd30f41135c16fc132e5b635988c162f353786ca28ec0605d4d8ed55799d5d02e9275473574bf3754975")
}

func (s *FingerprintSuite) TestString(c *gc.C) {
	raw := []byte("abcdefghijklmnopqrstuvwxyz1234567890123456789012")
	fp, err := resource.NewFingerprint(raw)
	c.Assert(err, jc.ErrorIsNil)
	hex := fp.String()

	c.Check(hex, jc.DeepEquals, "6162636465666768696a6b6c6d6e6f707172737475767778797a31323334353637383930313233343536373839303132")
}

func (s *FingerprintSuite) TestBytes(c *gc.C) {
	expected := []byte("abcdefghijklmnopqrstuvwxyz1234567890123456789012")
	fp, err := resource.NewFingerprint(expected)
	c.Assert(err, jc.ErrorIsNil)
	raw := fp.Bytes()

	c.Check(raw, jc.DeepEquals, expected)
}

func (s *FingerprintSuite) TestValidateOkay(c *gc.C) {
	fp, err := resource.NewFingerprint([]byte("123456789012345678901234567890123456789012345678"))
	c.Assert(err, jc.ErrorIsNil)
	err = fp.Validate()

	c.Check(err, jc.ErrorIsNil)
}

func (s *FingerprintSuite) TestValidateZero(c *gc.C) {
	var fp resource.Fingerprint
	err := fp.Validate()

	c.Check(err, jc.Satisfies, errors.IsNotValid)
	c.Check(err, gc.ErrorMatches, `zero-value fingerprint not valid`)
}
