package engine

import (
	"fmt"
	"os/exec"
	"strings"
)

type TriggerManager struct{}

func NewTriggerManager() *TriggerManager {
	return &TriggerManager{}
}

// Detecta el SSID actual del sistema (ejemplo usando nmcli en Linux)
func (t *TriggerManager) DetectCurrentSSID() string {
	cmd := exec.Command("nmcli", "-t", "-f", "ACTIVE,SSID", "dev", "wifi")
	output, err := cmd.Output()
	if err != nil {
		// Si no hay NetworkManager o falla, retornamos un valor por defecto o simulado
		return "Home_WiFi"
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 && parts[0] == "yes" {
			return parts[1]
		}
	}
	return "Unknown_Network"
}

// Evalúa si el disparador del manifiesto coincide con el entorno actual
func (t *TriggerManager) EvaluateTriggers(configuredTriggers map[string]string) string {
	expectedSSID, exists := configuredTriggers["network_ssid"]
	if !exists {
		return ""
	}

	currentSSID := t.DetectCurrentSSID()
	if currentSSID == expectedSSID {
		fmt.Printf("[edgelink-trigger] SSID detectado [%s] coincide con el ámbito configurado.\n", currentSSID)
		return "home"
	} else if currentSSID != "Unknown_Network" {
		fmt.Printf("[edgelink-trigger] SSID detectado [%s] difiere. Conmutando a ámbito seguro [transit].\n", currentSSID)
		return "transit"
	}

	return ""
}
