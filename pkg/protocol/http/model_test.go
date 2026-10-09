package http

import (
	"strings"
	"testing"
)

func TestOptsExtra_Checksum(t *testing.T) {
	extra := &OptsExtra{
		Checksum: &ChecksumOption{
			Algorithm: "sha256",
			Expected:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
	}
	if extra.Checksum.Algorithm != "sha256" || extra.Checksum.Expected != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected checksum: %+v", extra.Checksum)
	}
}

func TestChecksumOption_Validate(t *testing.T) {
	validMd5 := "0123456789abcdef0123456789abcdef"
	validSha1 := "0123456789abcdef0123456789abcdef01234567"
	validSha256 := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name      string
		option    *ChecksumOption
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "nil option",
			option:  nil,
			wantErr: false,
		},
		{
			name:    "empty option (empty strings)",
			option:  &ChecksumOption{Algorithm: "", Expected: ""},
			wantErr: false,
		},
		{
			name:    "empty option (whitespace strings)",
			option:  &ChecksumOption{Algorithm: "   ", Expected: "   "},
			wantErr: false,
		},
		{
			name:    "valid md5 lowercase",
			option:  &ChecksumOption{Algorithm: "md5", Expected: validMd5},
			wantErr: false,
		},
		{
			name:    "valid md5 uppercase",
			option:  &ChecksumOption{Algorithm: "MD5", Expected: strings.ToUpper(validMd5)},
			wantErr: false,
		},
		{
			name:    "valid sha1 lowercase",
			option:  &ChecksumOption{Algorithm: "sha1", Expected: validSha1},
			wantErr: false,
		},
		{
			name:    "valid sha-1 alias",
			option:  &ChecksumOption{Algorithm: "SHA-1", Expected: strings.ToUpper(validSha1)},
			wantErr: false,
		},
		{
			name:    "valid sha256 lowercase",
			option:  &ChecksumOption{Algorithm: "sha256", Expected: validSha256},
			wantErr: false,
		},
		{
			name:    "valid sha-256 alias",
			option:  &ChecksumOption{Algorithm: "sha-256", Expected: validSha256},
			wantErr: false,
		},
		{
			name:    "valid sha256 with whitespace around hash and algorithm",
			option:  &ChecksumOption{Algorithm: "  sha256  ", Expected: "  " + validSha256 + "  "},
			wantErr: false,
		},
		{
			name:      "unknown algorithm",
			option:    &ChecksumOption{Algorithm: "crc32", Expected: "12345678"},
			wantErr:   true,
			errSubstr: "unsupported checksum algorithm",
		},
		{
			name:      "algorithm set but hash blank",
			option:    &ChecksumOption{Algorithm: "sha256", Expected: "   "},
			wantErr:   true,
			errSubstr: "checksum expected hash is required",
		},
		{
			name:      "sha256 with abc (short length)",
			option:    &ChecksumOption{Algorithm: "sha256", Expected: "abc"},
			wantErr:   true,
			errSubstr: "expected 64 characters, got 3",
		},
		{
			name:      "sha256 with 64 x g (non-hex)",
			option:    &ChecksumOption{Algorithm: "sha256", Expected: strings.Repeat("g", 64)},
			wantErr:   true,
			errSubstr: "expected valid hexadecimal string",
		},
		{
			name:      "sha256 with 63 x a (off-by-one short)",
			option:    &ChecksumOption{Algorithm: "sha256", Expected: strings.Repeat("a", 63)},
			wantErr:   true,
			errSubstr: "expected 64 characters, got 63",
		},
		{
			name:      "sha256 with 65 x a (off-by-one long)",
			option:    &ChecksumOption{Algorithm: "sha256", Expected: strings.Repeat("a", 65)},
			wantErr:   true,
			errSubstr: "expected 64 characters, got 65",
		},
		{
			name:      "md5 with 31 chars",
			option:    &ChecksumOption{Algorithm: "md5", Expected: strings.Repeat("a", 31)},
			wantErr:   true,
			errSubstr: "expected 32 characters, got 31",
		},
		{
			name:      "md5 with 33 chars",
			option:    &ChecksumOption{Algorithm: "md5", Expected: strings.Repeat("a", 33)},
			wantErr:   true,
			errSubstr: "expected 32 characters, got 33",
		},
		{
			name:      "md5 with sha256-length hash (64 chars)",
			option:    &ChecksumOption{Algorithm: "md5", Expected: validSha256},
			wantErr:   true,
			errSubstr: "expected 32 characters, got 64",
		},
		{
			name:      "sha1 with 39 chars",
			option:    &ChecksumOption{Algorithm: "sha1", Expected: strings.Repeat("a", 39)},
			wantErr:   true,
			errSubstr: "expected 40 characters, got 39",
		},
		{
			name:      "sha1 with 41 chars",
			option:    &ChecksumOption{Algorithm: "sha1", Expected: strings.Repeat("a", 41)},
			wantErr:   true,
			errSubstr: "expected 40 characters, got 41",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.option.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errSubstr) {
					t.Fatalf("Validate() error = %v, want substring %q", err, tt.errSubstr)
				}
			}
		})
	}
}
