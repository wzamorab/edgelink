package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
)

type PodmanOrchestrator struct{}

func NewPodmanOrchestrator() *PodmanOrchestrator {
	return &PodmanOrchestrator{}
}

func (p *PodmanOrchestrator) EnsureServiceRunning(name string, port int, scope string) error {
	fmt.Printf("[edgelink-runtime] Verificando contenedor (Rootless): [%s] en puerto [%d] (Ámbito: %s)\n", name, port, scope)

	// Obtener el UID real (por defecto 1000 para el primer usuario del sistema)
	targetUID := os.Getenv("SUDO_UID")
	if targetUID == "" {
		targetUID = "1000"
	}

	// Obtener el nombre de usuario real de forma dinámica mediante su UID (evita hardcodear nombres)
	targetUser := os.Getenv("SUDO_USER")
	if targetUser == "" {
		if u, err := user.LookupId(targetUID); err == nil {
			targetUser = u.Username
		} else {
			targetUser = "root"
		}
	}

	// Comando para ejecutar Podman en el contexto rootless del usuario real
	runAsUser := func(args ...string) *exec.Cmd {
		envXdg := fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%s", targetUID)
		fullArgs := append([]string{"-u", targetUser, "env", envXdg, "podman"}, args...)
		return exec.Command("sudo", fullArgs...)
	}

	// Comprobar si el contenedor ya está corriendo
	checkCmd := runAsUser("inspect", "-f", "{{.State.Running}}", name)
	output, err := checkCmd.Output()
	if err == nil && string(output) == "true\n" {
		fmt.Printf(" -> Contenedor rootless [%s] ya se encuentra activo.\n", name)
		return nil
	}

	// Limpiar si existía detenido
	_ = runAsUser("rm", "-f", name).Run()

	// Lanzar el contenedor en entorno rootless del usuario
	deployCmd := runAsUser("run", "-d",
		"--name", name,
		"--publish", fmt.Sprintf("%d:%d", port, port),
		"--label", fmt.Sprintf("edgelink.scope=%s", scope),
		"docker.io/library/alpine:latest",
		"sh", "-c", "echo 'Servicio Edgelink Rootless E2EE Activo' && sleep infinity",
	)

	if err := deployCmd.Run(); err != nil {
		return fmt.Errorf("error al desplegar contenedor rootless como usuario %s: %w", targetUser, err)
	}

	fmt.Printf(" -> ¡Contenedor [%s] desplegado exitosamente en el espacio rootless de [%s]!\n", name, targetUser)
	return nil
}
