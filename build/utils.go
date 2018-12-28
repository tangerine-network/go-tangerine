// +build none

package build

import (
	"bytes"
	"encoding/base64"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
)

// GenerateEncodedPGPKeyPair generate base64 encoded PGP key pair
func GenerateEncodedPGPKeyPair(name, comment, email string) (string, error) {
	var e *openpgp.Entity
	e, err := openpgp.NewEntity(name, comment, email, nil)
	if err != nil {
		return "", err
	}

	keyBuffer := bytes.Buffer{}
	w, err := armor.Encode(&keyBuffer, openpgp.PrivateKeyType, nil)
	if err != nil {
		return "", err
	}

	err = e.SerializePrivate(w, nil)
	if err != nil {
		return "", err
	}
	w.Close()

	base64Buffer := bytes.Buffer{}
	en := base64.NewEncoder(base64.StdEncoding, &base64Buffer)
	en.Write(keyBuffer.Bytes())
	en.Close()

	return base64Buffer.String(), nil
}
