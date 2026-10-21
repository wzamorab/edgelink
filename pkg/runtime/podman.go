package runtime

import (
	"fmt"
	"os/exec"
	"os/user"
)

type PodmanOrchestrator struct{}

func NewPodmanOrchestrator() *PodmanOrchestrator {
	return &PodmanOrchestrator{}
}

func (p *PodmanOrchestrator) EnsureServiceRunning(name string, port int, scope string) error {
	fmt.Printf("[edgelink-runtime] Verificando contenedor (Rootless): [%s] en puerto [%d] (Ámbito: %s)\n", name, port, scope)

	// Obtenemos tu usuario real (wzamorab) para ejecutar Podman en su contexto rootless
	targetUser := "wzamorab"
	u, err := user.Lookup(targetUser)
	if err != nil {
		return fmt.Errorf("no se pudo encontrar al usuario %s: %w", targetUser, err)
	}

	// Comando base ejecutado a nombre de tu usuario real usando su UID y entorno
	// Esto obliga a Podman a correr en modo rootless puro aunque el demonio principal tenga sudo.
	runAsUser := func(args ...string) *exec.Cmd {
		fullArgs := append([]string{"-u", targetUser, "env", fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%s", u.Uid), "podman"}, args...)
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
		"sh", "-c", "echo 'Servicio Edgelink Rootless Puro Activo' && sleep infinity",
	)

	if err := deployCmd.Run(); err != nil {
		return fmt.Errorf("error al desplegar contenedor rootless como usuario %s: %w", targetUser, err)
	}

	fmt.Printf(" -> ¡Contenedor [%s] desplegado exitosamente en el espacio rootless de [%s]!\n", name, targetUser)
	return nil
}
