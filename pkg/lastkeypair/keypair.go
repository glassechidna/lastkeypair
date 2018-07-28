package lastkeypair

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"golang.org/x/crypto/ssh"
	"github.com/mitchellh/go-homedir"
	"path"
	"os"
	"io/ioutil"
	"crypto/rand"
	"github.com/pkg/errors"
)

type Keypair struct {
	PrivateKey []byte
	PublicKey []byte
}

func GenerateKeyPair() (*Keypair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, errors.Wrap(err, "generating privkey")
	}

	privateKeyDer := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privateKeyDer,
	}
	privateKeyPem := pem.EncodeToMemory(&privateKeyBlock)

	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "deriving pubkey from privkey")
	}

	pubkey := ssh.MarshalAuthorizedKey(pub)

	return &Keypair{
		PrivateKey: privateKeyPem,
		PublicKey: pubkey,
	}, nil
}

func AppDir() string {
	home, _ := homedir.Dir()
	appDir := path.Join(home, ".lkp")
	os.MkdirAll(appDir, 0755)
	return appDir
}

func TmpDir() string {
	tmpDir := path.Join(AppDir(), "tmp")
	os.MkdirAll(tmpDir, 0755)
	return tmpDir
}

func MyKeyPair() (*Keypair, error) {
	privkeyPath := path.Join(AppDir(), "id_rsa")
	pubkeyPath := path.Join(AppDir(), "id_rsa.pub")

	if _, err := os.Stat(privkeyPath); os.IsNotExist(err) {
		keypair, _ := GenerateKeyPair()
		ioutil.WriteFile(privkeyPath, keypair.PrivateKey, 0600)
		ioutil.WriteFile(pubkeyPath, keypair.PublicKey, 0644)
		return keypair, nil
	} else {
		keypair := Keypair{}
		keypair.PrivateKey, _ = ioutil.ReadFile(privkeyPath)
		keypair.PublicKey, _ = ioutil.ReadFile(pubkeyPath)
		return &keypair, nil
	}
}
