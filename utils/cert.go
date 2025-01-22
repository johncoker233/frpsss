package utils

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/credentials"
)

func PublicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func PemBlockForPrivKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			logrus.Fatalf("Unable to marshal ECDSA private key: %v", err)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

func TLSServerCert(certPem, keyPem []byte) (credentials.TransportCredentials, error) {
	cert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		return nil, err
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.NoClientCert,
	}
	return credentials.NewTLS(config), nil
}

func TLSClientCert(caPem []byte) (credentials.TransportCredentials, error) {
	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(caPem)
	return credentials.NewClientTLSFromCert(certpool, ""), nil
}

func TLSClientCertNoValidate(caPem []byte) (credentials.TransportCredentials, error) {
	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(caPem)

	config := &tls.Config{
		RootCAs:            certpool,
		InsecureSkipVerify: true,
	}

	return credentials.NewTLS(config), nil
}
