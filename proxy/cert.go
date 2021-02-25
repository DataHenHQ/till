// cert functions inspired by https://github.com/kr/mitm/cert

package proxy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"
)

const (
	caMaxAge   = 5 * 365 * 24 * time.Hour
	leafMaxAge = 24 * time.Hour
	caUsage    = x509.KeyUsageDigitalSignature |
		x509.KeyUsageContentCommitment |
		x509.KeyUsageKeyEncipherment |
		x509.KeyUsageDataEncipherment |
		x509.KeyUsageKeyAgreement |
		x509.KeyUsageCertSign |
		x509.KeyUsageCRLSign
	leafUsage = caUsage
)

// GenCert generates cert
func GenCert(names []string) (*tls.Certificate, error) {
	now := time.Now().Add(-1 * time.Hour).UTC()
	if !ca.Leaf.IsCA {
		return nil, errors.New("CA cert is not a CA")
	}
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %s", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{CommonName: names[0]},
		NotBefore:             now,
		NotAfter:              now.Add(leafMaxAge),
		KeyUsage:              leafUsage,
		BasicConstraintsValid: true,
		DNSNames:              names,
		SignatureAlgorithm:    x509.ECDSAWithSHA512,
	}
	key, err := genKeyPair()
	if err != nil {
		return nil, err
	}
	x, err := x509.CreateCertificate(rand.Reader, tmpl, ca.Leaf, key.Public(), ca.PrivateKey)
	if err != nil {
		return nil, err
	}
	cert := new(tls.Certificate)
	cert.Certificate = append(cert.Certificate, x)
	cert.PrivateKey = key
	cert.Leaf, _ = x509.ParseCertificate(x)
	return cert, nil
}

func genKeyPair() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
}

func GenCA(name string) (certPEM, keyPEM []byte, err error) {
	now := time.Now().UTC()
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             now,
		NotAfter:              now.Add(caMaxAge),
		KeyUsage:              caUsage,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            2,
		SignatureAlgorithm:    x509.ECDSAWithSHA512,
	}
	key, err := genKeyPair()
	if err != nil {
		return
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	if err != nil {
		return
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return
	}
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "ECDSA PRIVATE KEY",
		Bytes: keyDER,
	})

	return
}

// LoadOrGenCAFiles loads CA from file, or generates it into a file and use it
func LoadOrGenCAFiles(caCertFile, caKeyFile string) (err error) {
	var (
		caCertExists bool
		caKeyExists  bool
	)

	// check existense of the cert and key files
	if _, err := os.Stat(caCertFile); err == nil {
		caCertExists = true
	}
	if _, err := os.Stat(caKeyFile); err == nil {
		caKeyExists = true
	}

	// if both files exist, load from file
	if caCertExists && caKeyExists {
		err = loadCAVarFromFile(caCertFile, caKeyFile)
		if err != nil {
			return err
		}
		// loading certs message
		fmt.Println("Using the following Certificate Authority(CA) certificates:")
		fmt.Println("-", caCertFile)
		fmt.Println("-", caKeyFile)

		return nil
	}

	// if both does not exist, generate the key pair
	if !caCertExists && !caKeyExists {
		err = genCAToFile(caCertFile, caKeyFile)
		if err != nil {
			return err
		}
		err = loadCAVarFromFile(caCertFile, caKeyFile)
		if err != nil {

		}
		// generated certs message
		fmt.Println("Generated a new Certificate Authority(CA) certificates:")
		fmt.Println("-", caCertFile)
		fmt.Println("-", caKeyFile)
		return nil
	}

	// if one exist and not the other, then raise error
	if !caCertExists {
		log.Fatalln("ca-cert does not exist")
	}
	if !caKeyExists {
		log.Fatalln("ca-key does not exist")
	}

	return nil
}

// loadCAVarFromFile loads the keypair from file
func loadCAVarFromFile(caCertFile, caKeyFile string) (err error) {

	ca, err = tls.LoadX509KeyPair(caCertFile, caKeyFile)
	if err != nil {
		return err
	}

	ca.Leaf, err = x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		return err
	}

	return nil
}

// GenCAToFile generates the CA, and save it to files, and then use the ca
func genCAToFile(caCertFile string, caKeyFile string) (err error) {
	var hostname, _ = os.Hostname()

	certPEM, keyPEM, err := GenCA(hostname)
	if err != nil {
		log.Fatalln("Unable to generate CA", err)
	}

	if err := writeFullFilePath(caCertFile, certPEM, 0644); err != nil {
		log.Fatalln("Unable to write ca cert file to ", caCertFile)
	}

	if err := writeFullFilePath(caKeyFile, keyPEM, 0644); err != nil {
		log.Fatalln("Unable to write ca cert file to ", caCertFile)
	}

	return nil
}
