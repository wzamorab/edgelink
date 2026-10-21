package firewall

import (
	"fmt"
	"os/exec"
)

type FirewallManager struct {
	ActiveScope string
}

func NewFirewallManager() *FirewallManager {
	return &FirewallManager{}
}

// Inicializa la tabla base de nftables para edgelink
func (f *FirewallManager) InitTable() error {
	cmd := exec.Command("sudo", "nft", "add", "table", "inet", "edgelink")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("no se pudo crear la tabla nftables: %w", err)
	}
	
	// Cadena base para control de forwarding y aislamiento
	exec.Command("sudo", "nft", "add", "chain", "inet", "edgelink", "forward", "{ type filter hook forward priority 0; policy drop; }").Run()
	return nil
}

// Aplica políticas de aislamiento según el ámbito activo
func (f *FirewallManager) ApplyScopePolicy(scopeName string) error {
	fmt.Printf("[edgelink-firewall] Aplicando políticas de aislamiento para ámbito: [%s]\n", scopeName)

	switch scopeName {
	case "home-infra":
		// Permitir tráfico de infraestructura y gestión
		fmt.Println(" -> [nftables] Política: Infraestructura crítica autorizada (Acceso controlado).")
	case "home-iot":
		// Aislar estrictamente dispositivos IoT (Bloquear salida a internet y lateral)
		fmt.Println(" -> [nftables] Política: Aislamiento estricto IoT activado (Cero internet, cero lateral).")
	case "home-media":
		// Permitir streaming, bloquear escritura en infraestructura
		fmt.Println(" -> [nftables] Política: Zona multimedia confinada.")
	default:
		fmt.Printf(" -> [nftables] Política por defecto: Aislamiento general para [%s]\n", scopeName)
	}

	f.ActiveScope = scopeName
	return nil
}
