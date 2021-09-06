package cert

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
)

func SetupServerCert(namespaceName string, serviceName string) {
	// signingKey, err := createPrivateKey()
	// if err != nil {
	// 	log.Fatalf("Failed to create CA private key %v", err)
	// }

	// signingCert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: "quilkin-controller-cert"}, signingKey)
	// if err != nil {
	// 	log.Fatalf("Failed to create CA cert for apiserver %v", err)
	// }

	// caCertFile := filepath.Join("/tmp/signing-certs/", "ca.crt")

	// if err := ioutil.WriteFile(caCertFile, encodePEM(signingCert), 0644); err != nil {
	// 	log.Fatalf("Failed to write CA cert %v", err)
	// }

	key, err := createPrivateKey()
	if err != nil {
		log.Fatalf("Failed to create private key for %v", err)
	}

	signedCert, err := cert.NewSelfSignedCACert(
		cert.Config{
			CommonName: serviceName + "." + namespaceName + ".svc",
			AltNames:   cert.AltNames{DNSNames: []string{serviceName + "." + namespaceName + ".svc"}},
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		},
		key,
	)

	if err != nil {
		log.Fatalf("Failed to create cert%v", err)
	}
	err = os.Mkdir("/tmp/k8s-webhook-server/", 0755)
	err = os.Mkdir("/tmp/k8s-webhook-server/serving-certs/", 0755)
	if err != nil {
		log.Fatalf("Failed to create certs dir %v", err)
	}

	certFile := filepath.Join("/tmp/k8s-webhook-server/serving-certs/", "tls.crt")
	keyFile := filepath.Join("/tmp/k8s-webhook-server/serving-certs/", "tls.key")

	if err = ioutil.WriteFile(certFile, encodePEM(signedCert), 0600); err != nil {
		log.Fatalf("Failed to write cert file %v", err)
	}

	privateKeyPEM, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		log.Fatalf("Failed to marshal key %v", err)
	}

	if err = ioutil.WriteFile(keyFile, privateKeyPEM, 0644); err != nil {
		log.Fatalf("Failed to write key file %v", err)
	}
}

func createPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(cryptorand.Reader, 2048)
}

func encodePEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}
