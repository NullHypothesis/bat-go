package httpsignature

import (
	"crypto"
	"errors"
	"strconv"

	"golang.org/x/crypto/ed25519"
)

// Ed25519PubKey a wrapper type around ed25519.PublicKey to fulfill interface Verifier
type Ed25519PubKey ed25519.PublicKey

// Verify the signature sig for message using the ed25519 public key pk
// Returns true if the signature is valid, false if not and error if the key provided is not valid
func (pk Ed25519PubKey) Verify(message, sig []byte, opts crypto.SignerOpts) (bool, error) {
	if l := len(pk); l != ed25519.PublicKeySize {
		return false, errors.New("ed25519: bad public key length: " + strconv.Itoa(l))
	}

	key := make([]byte, ed25519.PublicKeySize)
	copy(key, pk)

	return ed25519.Verify(key, message, sig), nil
}
