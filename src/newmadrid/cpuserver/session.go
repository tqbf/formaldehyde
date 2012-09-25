package main

import (
	"io"
	"io/ioutil"
	"bytes"
	"encoding/json"
	"encoding/base64"
	"crypto/sha1"
	"crypto/aes"	
	"crypto/cipher"	
	"crypto/rand"
	"crypto/hmac"
	"crypto/subtle"
)

type Sessionkv map[string]string 

var (
	session_key string =	"photographers4510#bacteriologic"
	hmac_key string    =    "guaranty924]straightjacketing"
)

func NewSession() (*Sessionkv) {
	k := make(Sessionkv)
	return &k
}

func (s *Sessionkv) Encode() ([]byte) {
	return encode(s)
}

func (s *Sessionkv) Hash() ([]byte) {
	x, _ := json.Marshal(s)
	sha := sha1.New()
	sha.Write(x)
	return sha.Sum(nil)[0:4]	
}

func RestoreSession(raw []byte) *Sessionkv {
	return decode(raw)
}

func deriveKey(pass string) ([]byte) { 	
	for i := 0; i < 1000; i += 1 {	
		sha := sha1.New()
		sha.Write([]byte(pass))
		pass = base64.URLEncoding.EncodeToString(sha.Sum(nil))
	}

	return []byte(pass)
}

func cipherCore() (b cipher.Block) { 
 	b, _ = aes.NewCipher(deriveKey(session_key)[0:16])
	return
}

func randomBytes(n int) ([]byte) { 
	var buf []byte = make([]byte, n)
	n, err := rand.Read(buf[0:n])
	if err != nil { 
		panic("rand failure")
	}

	return buf[0:n]
}

func encryptor() (cipher.StreamWriter, []byte) { 
	iv := randomBytes(16)
	core := cipherCore()
	var b bytes.Buffer
	b.Write(iv)
	
	return cipher.StreamWriter{
		S: cipher.NewCTR(core, iv),
		W: &b,
	}, iv
}

func decryptor(iv []byte, ciphertext io.Reader) (cipher.StreamReader) { 
	core := cipherCore()
	return cipher.StreamReader{
		S: cipher.NewCTR(core, iv),
		R: ciphertext,
	}
}

func seal(ct []byte) ([]byte) { 
	var b bytes.Buffer

	hmac := hmac.New(sha1.New, deriveKey(hmac_key))
	_, err := hmac.Write(ct)
	if err != nil {
		panic("hmac error")
	}

	b.Write(ct)
	b.Write([]byte("---"))
	b.WriteString(base64.URLEncoding.EncodeToString(hmac.Sum(nil)))
	
	return b.Bytes()
} 

func verify(ct []byte) ([]byte) { 
	blobs := bytes.SplitN(ct, []byte("---"), 2)
	if len(blobs) != 2 {
		return nil
	}
	
	hmac := hmac.New(sha1.New, deriveKey(hmac_key))
	hmac.Write(blobs[0])

	valid := []byte(base64.URLEncoding.EncodeToString(hmac.Sum(nil)))

	if subtle.ConstantTimeCompare(blobs[1], valid) == 1 {
		return blobs[0]
	}

	return nil
}

func encode(session *Sessionkv) ([]byte) {
	x, _ := json.Marshal(session)
	e, _ := encryptor()
	e.Write(x)

	buf, _ := ioutil.ReadAll(e.W.(*bytes.Buffer))
	raw := seal(buf)

	return []byte(base64.URLEncoding.EncodeToString(raw))
}

func decode(raw []byte) (*Sessionkv) { 
	raw, _ = base64.URLEncoding.DecodeString(string(raw))
	raw = verify(raw)
	if raw == nil { 
		return nil
	}

	iv := raw[0:16]
	ct := raw[16:len(raw)]
	d := decryptor(iv, bytes.NewBuffer(ct))
	
	buf, _ := ioutil.ReadAll(d)
	var s Sessionkv

	err := json.Unmarshal(buf, &s)
	if err != nil { 
		return nil
	}

	return &s
}

func (self *Sessionkv) Map() map[string]string {
	return map[string]string(*self)
}