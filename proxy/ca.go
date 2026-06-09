package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

type CA struct {
	cert    *x509.Certificate
	key     *rsa.PrivateKey
	tlsCert tls.Certificate
}

func GenerateCA() (*CA, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Orbita CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, err
	}
	tlsCert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  key,
	}
	return &CA{
		cert:    cert,
		key:     key,
		tlsCert: tlsCert,
	}, nil
}

func (c *CA) Save(certPath, keyPath string) error {
	err := os.MkdirAll(filepath.Dir(certPath), 0755)
	if err != nil {
		return err
	}

	// cert file
	file, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer file.Close()

	err = pem.Encode(file, &pem.Block{Type: "CERTIFICATE", Bytes: c.tlsCert.Certificate[0]})
	if err != nil {
		return err
	}

	// key file
	kFile, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	defer kFile.Close()

	err = pem.Encode(kFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(c.key)})
	return err
}

func LoadCA(certPath, keyPath string) (*CA, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("invalid cert PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	certKey, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	keyBlock, _ := pem.Decode(certKey)
	if keyBlock == nil {
		return nil, fmt.Errorf("invalid key PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  key,
	}
	return &CA{
			cert,
			key,
			tlsCert,
		},
		nil
}

func LoadOrGenerate(certPath, keyPath string) (*CA, error) {
	ca, err := LoadCA(certPath, keyPath)
	if err == nil {
		return ca, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	ca, err = GenerateCA()
	if err != nil {
		return nil, err
	}

	return ca, ca.Save(certPath, keyPath)
}

func (c *CA) GenerateLeafCert(host string) (tls.Certificate, error) {
	hostName, _, err := net.SplitHostPort(host)
	if err != nil {
		hostName = host
	}
	serialBytes := make([]byte, 16)
	if _, err := rand.Read(serialBytes); err != nil {
		return tls.Certificate{}, err
	}
	serialNumber := new(big.Int).SetBytes(serialBytes)
	template := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{CommonName: hostName},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		DNSNames:              []string{hostName},
		IsCA:                  false,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, c.cert, &leafKey.PublicKey, c.key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{derBytes}, PrivateKey: leafKey}, nil
}
