package main

import (
	"fmt"
	"log"

	"github.com/wzamorab/edgelink/pkg/firewall"
	"github.com/wzamorab/edgelink/pkg/manifest"
	"github.com/wzamorab/edgelink/pkg/overlay"
	"github.com/wzamorab/edgelink/pkg/runtime"
)

func main() {
	fmt.Println("[edgelink] Iniciando motor de borde y reconciliación...")

	// 1. Cargar el manifiesto declarativo (Intención)
	m, err := manifest.LoadManifest("configs/sample.link.yaml")
	if err != nil {
		log.Fatalf("Error cargando manifiesto: %v", err)
	}
	fmt.Printf("[edgelink] Manifiesto cargado. Nodo ID: %s | Ámbito principal: %s\n", m.Node.ID, m.Node.Scope)

	// 2. Inicializar y aplicar políticas de Firewall basadas en la intención
	fw := firewall.NewFirewallManager()
	if err := fw.InitTable(); err != nil {
		log.Printf("Aviso de firewall (requiere sudo): %v", err)
	}
	if err := fw.ApplyScopePolicy(m.Node.Scope); err != nil {
		log.Printf("Error aplicando política de ámbito: %v", err)
	}

	// 3. Reconciliar la interfaz de red overlay (TUN) utilizando los valores declarados en el YAML
	ov := overlay.NewOverlayManager(m.Network.OverlayInterface)
	if err := ov.EnsureState(m.Network.Subnet); err != nil {
		log.Printf("Error asegurando estado del overlay TUN: %v", err)
	} else {
		fmt.Printf("[edgelink] Interfaz de red overlay [%s] reconciliada con la subred %s\n", m.Network.OverlayInterface, m.Network.Subnet)
	}

	// 4. Reconciliar el transporte UDP bidireccional y los Peers remotos declarados
	var mappedPeers []overlay.PeerRoute
	for _, p := range m.Network.Transport.Peers {
		mappedPeers = append(mappedPeers, overlay.PeerRoute{
			ID:         p.ID,
			Endpoint:   p.Endpoint,
			AllowedIPs: p.AllowedIPs,
		})
	}

	transport := overlay.NewUDPTransport(m.Network.Transport.ListenPort, mappedPeers, m.Network.OverlayInterface)
	if err := transport.Start(); err != nil {
		log.Printf("Error iniciando transporte UDP declarativo: %v", err)
	}

	// 5. Orquestar servicios declarados usando Podman rootless en el espacio del usuario
	podman := runtime.NewPodmanOrchestrator()
	for _, svc := range m.Services {
		if err := podman.EnsureServiceRunning(svc.Name, svc.Port, svc.Scope); err != nil {
			log.Printf("Error al asegurar servicio %s: %v", svc.Name, err)
		}
	}

	fmt.Println("[edgelink] Bucle de reconciliación completado exitosamente.")
}
