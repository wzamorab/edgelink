package main

import (
	"fmt"
	"log"
	"os"

	"github.com/wzamorab/edgelink/pkg/crypto"
	"github.com/wzamorab/edgelink/pkg/firewall"
	"github.com/wzamorab/edgelink/pkg/manifest"
	"github.com/wzamorab/edgelink/pkg/overlay"
	"github.com/wzamorab/edgelink/pkg/runtime"
	"gopkg.in/yaml.v3"
)

func main() {
	fmt.Println("[edgelink] Iniciando motor de borde y reconciliación...")

	yamlPath := "configs/sample.link.yaml"

	// 1. Cargar el manifiesto declarativo (Intención)
	m, err := manifest.LoadManifest(yamlPath)
	if err != nil {
		log.Fatalf("Error cargando manifiesto: %v", err)
	}

	// 1.1 BOOTSTRAP CRIPTOGRÁFICO: Si el nodo no tiene public_key en el YAML, la generamos y la auto-escribimos
	if m.Node.PublicKey == "" {
		fmt.Println("[edgelink-crypto] No se encontró llave pública declarada. Generando nueva identidad criptográfica...")
		identity, err := crypto.GenerateKeyPair()
		if err != nil {
			log.Fatalf("Error generando par de llaves: %v", err)
		}
		
		pubKeyHex := crypto.KeyToString(identity.PublicKey)
		m.Node.PublicKey = pubKeyHex
		fmt.Printf("[edgelink-crypto] ¡Llave generada con éxito! -> %s\n", pubKeyHex)

		// Reescribir el archivo YAML en disco con la nueva llave para mantener la intención declarativa sincronizada
		fileData, err := os.ReadFile(yamlPath)
		if err == nil {
			var rootNode yaml.Node
			if err := yaml.Unmarshal(fileData, &rootNode); err == nil {
				// Actualizamos de forma limpia guardando el archivo
				updatedData, err := yaml.Marshal(m)
				if err == nil {
					_ = os.WriteFile(yamlPath, updatedData, 0644)
					fmt.Println("[edgelink-crypto] Manifiesto YAML actualizado automáticamente con la public_key del nodo.")
				}
			}
		}
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

	// 3. Reconciliar la interfaz de red overlay (TUN)
	ov := overlay.NewOverlayManager(m.Network.OverlayInterface)
	if err := ov.EnsureState(m.Network.Subnet); err != nil {
		log.Printf("Error asegurando estado del overlay TUN: %v", err)
	} else {
		fmt.Printf("[edgelink] Interfaz de red overlay [%s] reconciliada con la subred %s\n", m.Network.OverlayInterface, m.Network.Subnet)
	}

	// 4. Mapear Peers y sus llaves públicas declaradas en el YAML
	var mappedPeers []overlay.PeerRoute
	for _, p := range m.Network.Transport.Peers {
		mappedPeers = append(mappedPeers, overlay.PeerRoute{
			ID:         p.ID,
			Endpoint:   p.Endpoint,
			PubKeyStr:  p.PublicKey,
			AllowedIPs: p.AllowedIPs,
		})
	}

	transport := overlay.NewUDPTransport(m.Network.Transport.ListenPort, mappedPeers, m.Network.OverlayInterface)
	if err := transport.Start(); err != nil {
		log.Printf("Error iniciando transporte UDP declarativo: %v", err)
	}

	// 5. Orquestar servicios declarados usando Podman rootless
	podman := runtime.NewPodmanOrchestrator()
	for _, svc := range m.Services {
		if err := podman.EnsureServiceRunning(svc.Name, svc.Port, svc.Scope); err != nil {
			log.Printf("Error al asegurar servicio %s: %v", svc.Name, err)
		}
	}

	fmt.Println("[edgelink] Bucle de reconciliación completado exitosamente.")
}
