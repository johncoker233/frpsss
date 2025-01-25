package dao

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"

	"fysj.net/v2/models"
	"fysj.net/v2/utils"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/credentials"
)

func InitCert(template *x509.Certificate) credentials.TransportCredentials {
	var (
		certPem []byte
		keyPem  []byte
	)
	cnt, err := CountCerts()
	if err != nil {
		logrus.Fatal(err)
	}
	if cnt == 0 {
		certPem, keyPem, err = GenX509Info(template)
		if err != nil {
			logrus.Fatal(err)
		}
		if err = models.GetDBManager().GetDefaultDB().Create(&models.Cert{
			Name:     "default",
			CertFile: certPem,
			CaFile:   certPem,
			KeyFile:  keyPem,
		}).Error; err != nil {
			logrus.Fatal(err)
		}
	} else {
		keyPem, certPem, err = GetDefaultKeyPair()
		if err != nil {
			logrus.Fatal(err)
		}
	}

	resp, err := utils.TLSServerCert(certPem, keyPem)
	if err != nil {
		logrus.Fatal(err)
	}
	return resp
}

func GenX509Info(template *x509.Certificate) (certPem []byte, keyPem []byte, err error) {

	priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	cert, err := x509.CreateCertificate(rand.Reader, template, template,
		priv.Public(), priv)
	if err != nil {
		return nil, nil, err
	}

	var certBuf bytes.Buffer
	pem.Encode(&certBuf, &pem.Block{
		Type: "CERTIFICATE", Bytes: cert,
	})

	var keyBuf bytes.Buffer
	pem.Encode(&keyBuf, utils.PemBlockForPrivKey(priv))
	return certBuf.Bytes(), keyBuf.Bytes(), nil
}

func CountCerts() (int64, error) {
	db := models.GetDBManager().GetDefaultDB()
	var count int64
	err := db.Model(&models.Cert{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetDefaultKeyPair() (keyPem []byte, certPem []byte, err error) {
	resp := &models.Cert{}
	err = models.GetDBManager().GetDefaultDB().Model(&models.Cert{}).
		Where(&models.Cert{Name: "default"}).First(resp).Error
	if err != nil {
		return nil, nil, err
	}
	return resp.KeyFile, resp.CertFile, nil
}
