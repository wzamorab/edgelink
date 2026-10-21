package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/nacl/box"
)

type NodeIdentity struct {
	PublicKey  *[32]byte
	PrivateKey *[32]byte
}

// Genera un par de llaves criptográficas para el nodo (Curva X25519 para cifrado)
func GenerateKeyPair() (*NodeIdentity, error) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("error generando llaves criptográficas: %w", err)
	}
	return &NodeIdentity{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// Cifra un mensaje para un Peer específico usando su llave pública
func EncryptPayload(message []byte, peerPublicKey *[32]byte, myPrivKey *[32]byte) ([]byte, [24]byte, error) {
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, nonce, err
	}

	encrypted := box.Seal(nil, message, &nonce, peerPublicKey, myPrivKey)
	return encrypted, nonce, nil
}

// Descifra un mensaje recibido de un Peer
func DecryptPayload(encrypted []byte, nonce [24]byte, peerPublicKey *[32]byte, myPrivKey *[32]byte) ([]byte, error) {
	decrypted, ok := box.Open(nil, encrypted, &nonce, peerPublicKey, myPrivKey)
	if !ok {
		return nil, fmt.Errorf("fallo al descifrar el paquete: autenticación criptográfica fallida")
	}
	return decrypted, nil
}

// Utilidad para convertir llaves a string hexadecimal (para logs o manifiestos)
func KeyToString(key *[32]byte) string {
	return hex.EncodeToString(key[:])
}
