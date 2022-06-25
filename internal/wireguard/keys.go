package wireguard

import (
	"encoding/base64"

	"github.com/h44z/wg-portal/internal/model"
	"github.com/pkg/errors"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func GetPrivateKeyBytes(p model.KeyPair) []byte {
	data, _ := base64.StdEncoding.DecodeString(p.PrivateKey)
	return data
}

func GetPublicKeyBytes(p model.KeyPair) []byte {
	data, _ := base64.StdEncoding.DecodeString(p.PublicKey)
	return data
}

func KeyBytesToString(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

type keyGenerator interface {
	GetFreshKeypair() (model.KeyPair, error)
	GetPreSharedKey() (model.PreSharedKey, error)
}

type wgCtrlKeyGenerator struct{}

func (k wgCtrlKeyGenerator) GetFreshKeypair() (model.KeyPair, error) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return model.KeyPair{}, errors.Wrap(err, "failed to generate private Key")
	}

	return model.KeyPair{
		PrivateKey: privateKey.String(),
		PublicKey:  privateKey.PublicKey().String(),
	}, nil
}

func (k wgCtrlKeyGenerator) GetPreSharedKey() (model.PreSharedKey, error) {
	preSharedKey, err := wgtypes.GenerateKey()
	if err != nil {
		return "", errors.Wrap(err, "failed to generate pre-shared Key")
	}

	return model.PreSharedKey(preSharedKey.String()), nil
}
