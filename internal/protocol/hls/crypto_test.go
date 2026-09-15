package hls

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"testing"
)

func pkcs7Pad(data []byte, blockSize int) []byte {
	pad := blockSize - len(data)%blockSize
	return append(data, bytes.Repeat([]byte{byte(pad)}, pad)...)
}

func TestDecryptAES128Roundtrip(t *testing.T) {
	key := make([]byte, 16)
	iv := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(iv); err != nil {
		t.Fatal(err)
	}
	plain := []byte("hello hls segment payload, padded to block size!")
	padded := pkcs7Pad(plain, aes.BlockSize)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	encrypted := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(encrypted, padded)

	got, err := decryptAES128(encrypted, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Errorf("roundtrip mismatch: %q", got)
	}
}

func TestDecryptAES128InvalidInput(t *testing.T) {
	if _, err := decryptAES128([]byte{1, 2, 3}, make([]byte, 16), make([]byte, 16)); err == nil {
		t.Error("non-block-aligned data should fail")
	}
	if _, err := decryptAES128(make([]byte, 16), make([]byte, 15), make([]byte, 16)); err == nil {
		t.Error("short key should fail")
	}
	if _, err := decryptAES128(make([]byte, 16), make([]byte, 16), make([]byte, 16)); err == nil {
		t.Error("all-zero block has invalid padding and should fail")
	}
}

func TestSequenceIV(t *testing.T) {
	iv := sequenceIV(1)
	if len(iv) != 16 || iv[15] != 1 {
		t.Errorf("sequence 1 IV wrong: %v", iv)
	}
	iv = sequenceIV(0x0102030405060708)
	for i, want := range []byte{1, 2, 3, 4, 5, 6, 7, 8} {
		if iv[8+i] != want {
			t.Errorf("sequence IV byte %d: want %d got %d", i, want, iv[8+i])
		}
	}
}
