package hls

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

// decryptAES128 decrypts an AES-128-CBC encrypted segment and strips PKCS7
// padding. data length must be a multiple of the AES block size.
func decryptAES128(data, key, iv []byte) ([]byte, error) {
	if len(key) != 16 {
		return nil, fmt.Errorf("invalid AES-128 key size %d", len(key))
	}
	if len(iv) != 16 {
		return nil, fmt.Errorf("invalid AES-128 IV size %d", len(iv))
	}
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("invalid AES-128 data size %d", len(data))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	plain := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, data)
	return pkcs7Unpad(plain, aes.BlockSize)
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty plaintext")
	}
	pad := int(data[len(data)-1])
	if pad == 0 || pad > blockSize || pad > len(data) {
		return nil, fmt.Errorf("invalid PKCS7 padding")
	}
	for _, b := range data[len(data)-pad:] {
		if int(b) != pad {
			return nil, fmt.Errorf("invalid PKCS7 padding")
		}
	}
	return data[:len(data)-pad], nil
}

// sequenceIV derives the default per-segment IV mandated by RFC 8216: the
// segment media sequence number as a 16-byte big-endian integer.
func sequenceIV(sequence int64) []byte {
	iv := make([]byte, 16)
	iv[8] = byte(sequence >> 56)
	iv[9] = byte(sequence >> 48)
	iv[10] = byte(sequence >> 40)
	iv[11] = byte(sequence >> 32)
	iv[12] = byte(sequence >> 24)
	iv[13] = byte(sequence >> 16)
	iv[14] = byte(sequence >> 8)
	iv[15] = byte(sequence)
	return iv
}
