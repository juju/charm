// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource_test

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v6-unstable/resource"
)

func newFingerprint(c *gc.C, data string) ([]byte, string) {
	hash := sha512.New384()
	_, err := hash.Write([]byte(data))
	c.Assert(err, jc.ErrorIsNil)
	raw := hash.Sum(nil)

	hexStr := hex.EncodeToString(raw)
	return raw, hexStr
}

var _ = gc.Suite(&FingerprintSuite{})

type FingerprintSuite struct{}

func (s *FingerprintSuite) TestNewFingerprintOkay(c *gc.C) {
	expected, _ := newFingerprint(c, "spamspamspam")

	fp, err := resource.NewFingerprint(expected)
	c.Assert(err, jc.ErrorIsNil)
	raw := fp.Bytes()

	c.Check(raw, jc.DeepEquals, expected)
}

func (s *FingerprintSuite) TestNewFingerprintTooSmall(c *gc.C) {
	expected, _ := newFingerprint(c, "spamspamspam")

	_, err := resource.NewFingerprint(expected[:10])

	c.Check(err, jc.Satisfies, errors.IsNotValid)
	c.Check(err, gc.ErrorMatches, `.*too small.*`)
}

func (s *FingerprintSuite) TestNewFingerprintTooBig(c *gc.C) {
	expected, _ := newFingerprint(c, "spamspamspam")

	_, err := resource.NewFingerprint(append(expected, 1, 2, 3))

	c.Check(err, jc.Satisfies, errors.IsNotValid)
	c.Check(err, gc.ErrorMatches, `.*too big.*`)
}

func (s *FingerprintSuite) TestGenerateFingerprint(c *gc.C) {
	expected, _ := newFingerprint(c, "spamspamspam")
	data := bytes.NewBufferString("spamspamspam")

	fp, err := resource.GenerateFingerprint(data)
	c.Assert(err, jc.ErrorIsNil)
	raw := fp.Bytes()

	c.Check(raw, jc.DeepEquals, expected)
}

func (s *FingerprintSuite) TestString(c *gc.C) {
	raw, expected := newFingerprint(c, "spamspamspam")
	fp, err := resource.NewFingerprint(raw)
	c.Assert(err, jc.ErrorIsNil)

	hex := fp.String()

	c.Check(hex, gc.Equals, expected)
}

func (s *FingerprintSuite) TestBytes(c *gc.C) {
	expected, _ := newFingerprint(c, "spamspamspam")
	fp, err := resource.NewFingerprint(expected)
	c.Assert(err, jc.ErrorIsNil)

	raw := fp.Bytes()

	c.Check(raw, jc.DeepEquals, expected)
}

func (s *FingerprintSuite) TestValidateOkay(c *gc.C) {
	raw, _ := newFingerprint(c, "spamspamspam")
	fp, err := resource.NewFingerprint(raw)
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
