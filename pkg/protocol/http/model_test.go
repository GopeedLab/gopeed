package http

import "testing"

func TestOptsExtra_Checksum(t *testing.T) {
	extra := &OptsExtra{
		Checksum: &ChecksumOption{
			Algorithm: "sha256",
			Expected:  "abcdef1234567890",
		},
	}
	if extra.Checksum.Algorithm != "sha256" || extra.Checksum.Expected != "abcdef1234567890" {
		t.Fatalf("unexpected checksum: %+v", extra.Checksum)
	}
}
