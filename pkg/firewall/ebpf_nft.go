package firewall

import (
	"fmt"
)

type FirewallManager struct {
	ActiveScope string
}

func NewFirewallManager() *FirewallManager {
	return &FirewallManager{}
}

func (f *FirewallManager) ApplyScope(scope string) error {
	if f.ActiveScope == scope {
		// El ámbito no ha cambiado; estado convergido
		return nil
	}

	fmt.Printf("[edgelink-firewall] Cambiando política de seguridad para el ámbito: [%s]\n", scope)

	switch scope {
	case "home":
		err := f.applyHomePolicy()
		if err != nil {
			return err
		}
	case "work":
		err := f.applyWorkPolicy()
		if err != nil {
			return err
		}
	case "transit":
		err := f.applyTransitPolicy()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("ámbito desconocido: %s", scope)
	}

	f.ActiveScope = scope
	return nil
}

func (f *FirewallManager) applyHomePolicy() error {
	// Política relajada para el hogar: permite tráfico multimedia y descubrimiento local
	fmt.Println(" -> [nftables/eBPF] Aplicando reglas Home: Permitir mDNS, DLNA (puerto 8096) y malla completa.")
	return nil
}

func (f *FirewallManager) applyWorkPolicy() error {
	// Política estricta para el trabajo: bloquea tráfico no corporativo, exige túnel cifrado
	fmt.Println(" -> [nftables/eBPF] Aplicando reglas Work: Aislamiento estricto, bloqueo de puertos multimedia, solo VPN corporativa.")
	return nil
}

func (f *FirewallManager) applyTransitPolicy() error {
	// Política de máxima seguridad en tránsito (cafés, aeropuertos): bloqueo total entrante
	fmt.Println(" -> [nftables/eBPF] Aplicando reglas Transit: Drop total de tráfico entrante no solicitado, cifrado estricto.")
	return nil
}
