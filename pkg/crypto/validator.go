package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
)

type Validator struct {
	trustedPublicKeys map[string]ed25519.PublicKey
}

func NewValidator() *Validator {
	return &Validator{
		trustedPublicKeys: make(map[string]ed25519.PublicKey),
	}
}

// Registra la llave pública (en formato hex) autorizada para una federación
func (v *Validator) RegisterFederationKey(federation string, pubKeyHex string) error {
	bytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return err
	}
	if len(bytes) != ed25519.PublicKeySize {
		return fmt.Errorf("tamaño de llave pública Ed25519 inválido")
	}
	v.trustedPublicKeys[federation] = ed25519.PublicKey(bytes)
	return nil
}

// Valida una firma sobre el contenido del manifiesto
func (v *Validator) Verify(federation string, manifestData []byte, signatureHex string) bool {
	pubKey, exists := v.trustedPublicKeys[federation]
	if !exists {
		// Si no hay llave registrada para la federación, por seguridad estricta se rechaza o se emite alerta
		return false
	}

	sig, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}

	return ed25519.Verify(pubKey, manifestData, sig)
}
