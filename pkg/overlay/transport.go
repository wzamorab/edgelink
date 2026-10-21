package overlay

import (
	"fmt"
	"net"
	"os"
)

type PeerRoute struct {
	ID         string
	Endpoint   string
	AllowedIPs []string
}

type UDPTransport struct {
	ListenPort       int
	Peers            []PeerRoute
	OverlayInterface string
	udpConn          *net.UDPConn
	tunFile          *os.File
}

func NewUDPTransport(port int, peers []PeerRoute, overlayInterface string) *UDPTransport {
	return &UDPTransport{
		ListenPort:       port,
		Peers:            peers,
		OverlayInterface: overlayInterface,
	}
}

func (t *UDPTransport) Start() error {
	// 1. Abrir el archivo del dispositivo TUN en el kernel (/dev/net/tun)
	// Nota: Esto requiere que la interfaz ya haya sido creada por netlink.
	tunPath := "/dev/net/tun"
	file, err := os.OpenFile(tunPath, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("no se pudo abrir %s: %w", tunPath, err)
	}
	t.tunFile = file

	// 2. Iniciar el listener UDP
	lAddr, err := net.ResolveUDPAddr("udp6", fmt.Sprintf("[::]:%d", t.ListenPort))
	if err != nil {
		return fmt.Errorf("error resolviendo dirección UDP: %w", err)
	}

	conn, err := net.ListenUDP("udp6", lAddr)
	if err != nil {
		return fmt.Errorf("error iniciando servidor UDP en puerto %d: %w", t.ListenPort, err)
	}
	t.udpConn = conn

	fmt.Printf("[edgelink-transport] Red declarativa activa. Escuchando UDP en puerto [::]:%d\n", t.ListenPort)

	for _, peer := range t.Peers {
		fmt.Printf(" -> [Reconciliador P2P] Peer registrado -> ID: %s | Endpoint: %s | IPs: %v\n", 
			peer.ID, peer.Endpoint, peer.AllowedIPs)
	}

	// 3. Iniciar bucles de reenvío bidireccional
	go t.readUDPToTUN()
	go t.readTUNToUDP()

	return nil
}

// Recibe paquetes UDP de la red y los inyecta en la interfaz TUN local
func (t *UDPTransport) readUDPToTUN() {
	buf := make([]byte, 1500)
	for {
		n, remoteAddr, err := t.udpConn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		if t.tunFile != nil {
			_, _ = t.tunFile.Write(buf[:n])
			fmt.Printf("[edgelink-transport] Paquete P2P recibido de %s (%d bytes) -> Inyectado en TUN\n", remoteAddr.String(), n)
		}
	}
}

// Lee paquetes que el kernel envía a la interfaz TUN y los encapsula por UDP hacia el peer
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

		// Si hay peers declarados, enviamos el paquete al primer endpoint configurado
		if len(t.Peers) > 0 && t.Peers[0].Endpoint != "" {
			rAddr, err := net.ResolveUDPAddr("udp", t.Peers[0].Endpoint)
			if err != nil {
				continue
			}
			_, _ = t.udpConn.WriteToUDP(buf[:n], rAddr)
		}
	}
}
