//     Copyright (C) 2020-2021, IrineSistiana
//
//     This file is part of simple-tls.
//
//     simple-tls is free software: you can redistribute it and/or modify
//     it under the terms of the GNU General Public License as published by
//     the Free Software Foundation, either version 3 of the License, or
//     (at your option) any later version.
//
//     simple-tls is distributed in the hope that it will be useful,
//     but WITHOUT ANY WARRANTY; without even the implied warranty of
//     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//     GNU General Public License for more details.
//
//     You should have received a copy of the GNU General Public License
//     along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/chunqian/simple-tls/core/mlog"
	"math/big"
	mathRand "math/rand"
	"os"
	"time"
)

var logger = mlog.L()

func GenerateCertificate(serverName string, template *x509.Certificate) (dnsName string, cert *x509.Certificate, keyPEM, certPEM []byte, err error) {
	//priv key
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return
	}

	// set DNSNames
	if len(serverName) == 0 {
		dnsName = randServerName()
	} else {
		dnsName = serverName
	}

	if template == nil {
		//serial number
		serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
		var serialNumber *big.Int
		serialNumber, err = rand.Int(rand.Reader, serialNumberLimit)
		if err != nil {
			err = fmt.Errorf("generate serial number: %v", err)
			return
		}

		template = &x509.Certificate{
			SerialNumber: serialNumber,
			Subject:      pkix.Name{CommonName: dnsName},
			DNSNames:     []string{dnsName},
			NotBefore:    time.Now(),

			KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			BasicConstraintsValid: true,
		}
	} else {
		if len(template.DNSNames) > 0 {
			dnsName = template.DNSNames[0]
		}
	}
	template.NotAfter = time.Now().AddDate(10, 0, 0)
	template.SignatureAlgorithm = x509.UnknownSignatureAlgorithm
	template.PublicKey = nil

	parent := &x509.Certificate{
		SerialNumber: new(big.Int),
		Subject:      template.Issuer,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, key)
	if err != nil {
		return
	}
	template.NotAfter = time.Now().AddDate(10, 0, 0)
	template.SignatureAlgorithm = x509.UnknownSignatureAlgorithm
	template.PublicKey = nil

	b, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	cert, err = x509.ParseCertificate(certDER)
	if err != nil {
		return
	}

	return
}

func randServerName() string {
	r := mathRand.New(mathRand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%s.%s", randStr(r.Intn(5)+3, r), randStr(r.Intn(3)+1, r))
}

func randStr(length int, r *mathRand.Rand) string {
	set := "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = set[r.Intn(len(set))]
	}
	return string(b)
}

func LoadCert(file string) (*x509.Certificate, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("empty data")
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("invalid pem block type [%s]", block.Type)
	}
	return x509.ParseCertificate(block.Bytes)
}
