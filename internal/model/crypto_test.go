package model

import (
	"testing"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/stretchr/testify/assert"
)

func TestKeyBytesToString(t *testing.T) {
	assert.Equal(t, "aGVsbG8=", KeyBytesToString([]byte("hello")))
}

func TestKeyPair_GetPrivateKey(t *testing.T) {
	kp := KeyPair{
		PrivateKey: "YClVPv8kGwovpNgNwxRRHgALG6QtPbu78uhRijI1wmU=",
		PublicKey:  "5EoviOwRzZFAzVzgzKIgifdd0FaijuUVX3vDbmpUlxQ=",
	}

	got := kp.GetPrivateKey()
	want, _ := wgtypes.ParseKey("YClVPv8kGwovpNgNwxRRHgALG6QtPbu78uhRijI1wmU=")
	assert.Equal(t, want, got)
}

func TestKeyPair_GetPrivateKeyBytes(t *testing.T) {
	kp := KeyPair{
		PrivateKey: "aGVsbG8=",
		PublicKey:  "d29ybGQ=",
	}

	got := kp.GetPrivateKeyBytes()
	assert.Equal(t, []byte("hello"), got)
}

func TestKeyPair_GetPublicKey(t *testing.T) {
	kp := KeyPair{
		PrivateKey: "YClVPv8kGwovpNgNwxRRHgALG6QtPbu78uhRijI1wmU=",
		PublicKey:  "5EoviOwRzZFAzVzgzKIgifdd0FaijuUVX3vDbmpUlxQ=",
	}

	got := kp.GetPublicKey()
	want, _ := wgtypes.ParseKey("5EoviOwRzZFAzVzgzKIgifdd0FaijuUVX3vDbmpUlxQ=")
	assert.Equal(t, want, got)
}

func TestKeyPair_GetPublicKeyBytes(t *testing.T) {
	kp := KeyPair{
		PrivateKey: "aGVsbG8=",
		PublicKey:  "d29ybGQ=",
	}

	got := kp.GetPublicKeyBytes()
	assert.Equal(t, []byte("world"), got)
}

func TestNewFreshKeypair(t *testing.T) {
	kp, err := NewFreshKeypair()
	assert.NoError(t, err)
	assert.NotEmpty(t, kp.PrivateKey)
	assert.NotEmpty(t, kp.PublicKey)
}

func TestNewPreSharedKey(t *testing.T) {
	psk, err := NewPreSharedKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, psk)
}
