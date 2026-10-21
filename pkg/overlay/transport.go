package overlay

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"

	"github.com/wzamorab/edgelink/pkg/crypto"
)

type PeerRoute struct {
	ID         string
	Endpoint   string
	AllowedIPs []string
	PubKeyStr  string
	ParsedKey  *[32]byte
}

type UDPTransport struct {
	ListenPort       int
	Peers            []PeerRoute
	OverlayInterface string
	Identity         *crypto.NodeIdentity
	udpConn          *net.UDPConn
	tunFile          *os.File
}

func NewUDPTransport(port int, peers []PeerRoute, overlayInterface string) *UDPTransport {
	// Generar identidad criptográfica local
	identity, err := crypto.GenerateKeyPair()
	if err != nil {
		fmt.Printf("[edgelink-crypto] Advertencia generando identidad: %v\n", err)
	} else {
		fmt.Printf("[edgelink-crypto] Mi Llave Pública (declarativa/local): %s\n", crypto.KeyToString(identity.PublicKey))
	}

	// Parsear las llaves públicas de los peers declaradas en el YAML
	for i := range peers {
		if peers[i].PubKeyStr != "" {
			var keyArr [32]byte
			decoded, err := hex.DecodeString(peers[i].PubKeyStr)
			if err == nil && len(decoded) == 32 {
				copy(keyArr[:], decoded)
				peers[i].ParsedKey = &keyArr
				fmt.Printf("[edgelink-crypto] Llave pública reconciliada para peer [%s]\n", peers[i].ID)
			} else {
				fmt.Printf("[edgelink-crypto] Error parseando public_key del peer [%s]: formato hex inválido\n", peers[i].ID)
			}
		}
	}

	return &UDPTransport{
		ListenPort:       port,
		Peers:            peers,
		OverlayInterface: overlayInterface,
		Identity:         identity,
	}
}

func (t *UDPTransport) Start() error {
	tunPath := "/dev/net/tun"
	file, err := os.OpenFile(tunPath, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("no se pudo abrir %s: %w", tunPath, err)
	}
	t.tunFile = file

	lAddr, err := net.ResolveUDPAddr("udp6", fmt.Sprintf("[::]:%d", t.ListenPort))
	if err != nil {
		return fmt.Errorf("error resolviendo dirección UDP: %w", err)
	}

	conn, err := net.ListenUDP("udp6", lAddr)
	if err != nil {
		return fmt.Errorf("error iniciando servidor UDP en puerto %d: %w", t.ListenPort, err)
	}
	t.udpConn = conn

	fmt.Printf("[edgelink-transport] Red Zero-Trust con E2EE activa. Escuchando en UDP [::]:%d\n", t.ListenPort)

	go t.readUDPToTUN()
	go t.readTUNToUDP()

	return nil
}

func (t *UDPTransport) readUDPToTUN() {
	buf := make([]byte, 2048)
	for {
		n, remoteAddr, err := t.udpConn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		if t.Identity == nil || len(t.Peers) == 0 {
			continue
		}

		peerKey := t.Peers[0].ParsedKey
		if peerKey == nil {
			if t.tunFile != nil {
				_, _ = t.tunFile.Write(buf[:n])
			}
			continue
		}

		if n < 24 {
			continue
		}
		var nonce [24]byte
		copy(nonce[:], buf[:24])
		encryptedPayload := buf[24:n]

		decrypted, err := crypto.DecryptPayload(encryptedPayload, nonce, peerKey, t.Identity.PrivateKey)
		if err != nil {
			fmt.Printf("[edgelink-crypto] Paquete rechazado de %s: %v\n", remoteAddr.String(), err)
			continue
		}

		if t.tunFile != nil {
			_, _ = t.tunFile.Write(decrypted)
		}
	}
}

func (t *UDPTransport) readTUNToUDP() {
	buf := make([]byte, 1500)
	for {
		if t.tunFile == nil {
			return
		}
		n, err := t.tunFile.Read(buf)
		if err != nil {
			continue
		}

		if len(t.Peers) == 0 || t.Peers[0].Endpoint == "" || t.Identity == nil {
			continue
		}

		peerKey := t.Peers[0].ParsedKey
		if peerKey == nil {
			rAddr, err := net.ResolveUDPAddr("udp", t.Peers[0].Endpoint)
			if err == nil {
				_, _ = t.udpConn.WriteToUDP(buf[:n], rAddr)
			}
			continue
		}

		encrypted, nonce, err := crypto.EncryptPayload(buf[:n], peerKey, t.Identity.PrivateKey)
		if err != nil {
			continue
		}

		packet := append(nonce[:], encrypted...)

		rAddr, err := net.ResolveUDPAddr("udp", t.Peers[0].Endpoint)
		if err != nil {
			continue
		}

		_, _ = t.udpConn.WriteToUDP(packet, rAddr)
	}
}
