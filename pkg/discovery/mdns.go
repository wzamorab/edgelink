package discovery

import (
	"fmt"
	"net"
	"time"
)

type NodeDiscovery struct {
	NodeID     string
	ListenPort int
	MulticastAddr string
}

func NewNodeDiscovery(nodeID string, port int) *NodeDiscovery {
	return &NodeDiscovery{
		NodeID:        nodeID,
		ListenPort:    port,
		MulticastAddr: "[ff02::1]:53530", // Dirección IPv6 multicast local para edgelink
	}
}

// Inicia el servicio de descubrimiento en segundo plano
func (d *NodeDiscovery) Start() {
	go d.advertiseLoop()
	go d.listenLoop()
	fmt.Printf("[edgelink-discovery] Módulo de auto-descubrimiento activo para el nodo [%s]\n", d.NodeID)
}

// Anuncia periódicamente la presencia del nodo en la red local
func (d *NodeDiscovery) advertiseLoop() {
	addr, err := net.ResolveUDPAddr("udp6", d.MulticastAddr)
	if err != nil {
		return
	}

	conn, err := net.DialUDP("udp6", nil, addr)
	if err != nil {
		return
	}
	defer conn.Close()

	message := fmt.Sprintf("EDGELINK_ANNOUNCE:%s:%d", d.NodeID, d.ListenPort)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		_, _ = conn.Write([]byte(message))
	}
}

// Escucha anuncios de otros nodos en la red local
func (d *NodeDiscovery) listenLoop() {
	addr, err := net.ResolveUDPAddr("udp6", "[::]:53530")
	if err != nil {
		return
	}

	conn, err := net.ListenUDP("udp6", addr)
	if err != nil {
		return
	}
	defer conn.Close()

	buf := make([]byte, 512)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		announcement := string(buf[:n])
		// Procesar el anuncio (aquí detectaría peers nuevos automáticamente)
		if len(announcement) > 18 && announcement[:18] == "EDGELINK_ANNOUNCE:" {
			fmt.Printf("[edgelink-discovery] ¡Peer detectado en la red local! Desde %s -> %s\n", remoteAddr.IP.String(), announcement)
		}
	}
}
