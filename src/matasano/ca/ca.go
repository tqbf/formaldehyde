package ca

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"matasano/util"
	"math/big"
	"time"
)

func ConvertDERToPem(der []byte, label string) []byte {
	outbuf := bytes.NewBuffer(make([]byte, 0, 100))
	outbuf.Write([]byte(fmt.Sprintf("-----BEGIN %s-----\n", label)))
	encoder := base64.NewEncoder(base64.StdEncoding, outbuf)
	encoder.Write(der)
	encoder.Close()
	outbuf.Write([]byte(fmt.Sprintf("\n-----END %s-----\n", label)))
	return outbuf.Bytes()
}

func randint() int64 {
	r, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	return r.Int64()
}

func Test() { 
	// do as i say not as i do
	ca, err := NewCA("/tmp/", "/tmp/cert.der", "/tmp/key.der")
	fmt.Println(ca, err)
	cert, err := ca.Lookup("www.foo.com")
	fmt.Println(cert, err)
}

func simple_name(cn string) pkix.Name {
	subject := pkix.Name{}
	subject.Country = append(subject.Country, "US")
	subject.Organization = append(subject.Organization, "Matasano Security")
	subject.CommonName = cn
	return subject
}

type CA struct {
	cache_path string
	cache	   map[string] *tls.Certificate
	cert_path  string
	key_path   string
	key        *rsa.PrivateKey
	cert       *x509.Certificate
}

func (self *CA) init() (err error) {
	var b *bytes.Buffer
	if b, err = util.Slurp(self.key_path); err != nil { 
		return 
	}
	if self.key, err = x509.ParsePKCS1PrivateKey(b.Bytes()); err != nil { 
		return 
	}
	if b, err = util.Slurp(self.cert_path); err != nil { 
		return 
	}
	if self.cert, err = x509.ParseCertificate(b.Bytes()); err != nil { 
		return 
	}

	return nil
}

func NewCA(cache_path, cert_path string, key_path string) (*CA, error) {
	ca := CA{
		cache_path: cache_path,
		cert_path:  cert_path,
		key_path:   key_path,
		cache:      make(map[string] *tls.Certificate),
	}

	if err := ca.init(); err != nil { 
		return nil, err
	}

	return &ca, nil
}

func (self *CA) Mint(cn string) (cert *tls.Certificate, err error) {
	cert = nil
	ser := big.NewInt(time.Now().Unix() + 100 + int64(randint()))

	tmpl := &x509.Certificate{
		SerialNumber: ser,
		Subject:      simple_name(cn),
		NotBefore:    time.Unix(time.Now().Unix()-10000, 0),
		NotAfter:     time.Unix(int64(time.Now().Unix()+(100*24*50*60*60)), 0),
		KeyUsage:     x509.KeyUsageDataEncipherment,
		SubjectKeyId: self.cert.SubjectKeyId,
	}

	private_key, err := rsa.GenerateKey(rand.Reader, 1024)
	certificate, err := x509.CreateCertificate(rand.Reader, tmpl, self.cert, &private_key.PublicKey, self.key)
	if err != nil {
		return
	}

	keyder := x509.MarshalPKCS1PrivateKey(private_key)
	self.to_disk(cn, keyder, certificate)

	tls_cert, err := tls.X509KeyPair(
		ConvertDERToPem(certificate, "CERTIFICATE"),
		ConvertDERToPem(keyder, "PRIVATE KEY"))
	if err != nil {
		return
	}

	cert = &tls_cert
	return
}

func (self *CA) to_disk(cn string, keyder, certder []byte) (err error) {
	err = util.Barf(fmt.Sprintf("%s/%s.key", self.cache_path, cn), keyder)
	if err != nil {
		return
	}

	err = util.Barf(fmt.Sprintf("%s/%s.crt", self.cache_path, cn), certder)
	return
}

func (self *CA) from_disk(cn string) (cert *tls.Certificate, err error) {
	kblob, err := util.Slurp(fmt.Sprintf("%s/%s.key", self.cache_path, cn))
	if(err != nil) {
		return
	}

	cblob, err := util.Slurp(fmt.Sprintf("%s/%s.crt", self.cache_path, cn))
	if(err != nil) {
		return
	}

	tcert, err := tls.X509KeyPair(
		ConvertDERToPem(cblob.Bytes(), "CERTIFICATE"),
		ConvertDERToPem(kblob.Bytes(), "PRIVATE KEY"))
	cert = &tcert
	return
}

func (self *CA) Lookup(cn string) (cert *tls.Certificate, err error) {
	cert, ok := self.cache[cn]
	if ok {
		return
	}

	cert, err = self.from_disk(cn)
	if err == nil {
		return
	}

	cert, err = self.Mint(cn)
	if err != nil { 
		return
	}

	self.cache[cn] = cert
	return
}

