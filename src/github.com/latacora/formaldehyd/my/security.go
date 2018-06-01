package my

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/scrypt"
)

// A PasswordHash is anything with a Validate message that returns true
// or false based on whether a password is valid.

type PasswordHash interface {
	Validate(password string) bool
}

type ScryptHash struct {
	Hash []byte
	Salt []byte
}

// ScryptHashFromPassword returns an ScryptHash, which is a PasswordHash,
// based on a password.

func ScryptHashFromPassword(password string) *ScryptHash {
	salt := CryptoRandBytes(8)
	key, err := scrypt.Key([]byte(password), salt, 16384, 8, 1, 32)
	if err != nil {
		panic("unexpected scrypt error")
	}

	return &ScryptHash{
		Hash: key,
		Salt: salt,
	}
}

// Encode turns an ScryptHash into something you can store in a database
// string column.

func (s *ScryptHash) Encode() string {
	return fmt.Sprintf("%s:%s", hex.EncodeToString(s.Salt), hex.EncodeToString(s.Hash))
}

// ScryptHashFromHashString takes the string output of Encode and turns
// it back into a PasswordHash

func ScryptHashFromHashString(hash string) (ret *ScryptHash, err error) {
	tup := strings.Split(hash, ":")
	if len(tup) != 2 {
		return nil, fmt.Errorf("invalid hash")
	}

	ret = &ScryptHash{}
	ret.Salt, err = hex.DecodeString(tup[0])
	if err != nil {
		return
	}

	ret.Hash, err = hex.DecodeString(tup[1])
	return
}

// Validate takes a password and returns whether or not it matches the
// stored password

func (s *ScryptHash) Validate(password string) bool {
	h, err := scrypt.Key([]byte(password), s.Salt, 16384, 8, 1, 32)
	if err != nil {
		panic("unexpected scrypt error")
	}

	return bytes.Equal(h, s.Hash)
}

// CryptoRand64 generates a 64 bit random number, without the inconvience
// of roundtripping it through a bignum.

func CryptoRand64() (ret uint64) {
	var buf [8]byte

	if _, err := io.ReadFull(rand.Reader, buf[:]); err != nil {
		panic("partial read from urandom")
	}

	for i := 0; i < 8; i++ {
		ret <<= 8
		ret |= uint64(buf[i])
	}

	return
}

// CryptoRandBytes returns an n-byte slice of random bytes.

func CryptoRandBytes(n int) (ret []byte) {
	ret = make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, ret); err != nil {
		panic("partial read from urandom")
	}
	return ret
}
