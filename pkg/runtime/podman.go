package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type PodmanManager struct{}

func NewPodmanManager() *PodmanManager {
	return &PodmanManager{}
}

type ServiceIntent struct {
	Name   string
	Status string
	Port   int
}

// Helper para construir comandos de podman ejecutados como el usuario real (rootless)
func (p *PodmanManager) podmanCommand(args ...string) *exec.Cmd {
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" && sudoUser != "root" {
		// Si se ejecutó con sudo, ejecutamos podman como el usuario original con su XDG_RUNTIME_DIR
		fullArgs := append([]string{"-u", sudoUser, "podman"}, args...)
		cmd := exec.Command("sudo", fullArgs...)
		cmd.Env = append(os.Environ(), fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%s", getUID(sudoUser)))
		return cmd
	}
	return exec.Command("podman", args...)
}

func getUID(username string) string {
	cmd := exec.Command("id", "-u", username)
	out, err := cmd.Output()
	if err != nil {
		return "1000" // Fallback común
	}
	return strings.TrimSpace(string(out))
}

func (p *PodmanManager) ReconcileService(svc ServiceIntent, overlaySubnet string) error {
	isRunning, err := p.isContainerRunning(svc.Name)
	if err != nil {
		return fmt.Errorf("fallo al consultar estado de Podman para %s: %v", svc.Name, err)
	}

	if svc.Status == "running" && !isRunning {
		fmt.Printf("[edgelink-runtime] Contenedor [%s] no está ejecutándose. Iniciando en modo rootless...\n", svc.Name)
		return p.startContainer(svc)
	} else if svc.Status == "stopped" && isRunning {
		fmt.Printf("[edgelink-runtime] Contenedor [%s] debe estar detenido. Deteniendo...\n", svc.Name)
		return p.stopContainer(svc.Name)
	}

	return nil
}

func (p *PodmanManager) isContainerRunning(name string) (bool, error) {
	cmd := p.podmanCommand("ps", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	runningNames := strings.TrimSpace(string(output))
	return runningNames == name, nil
}

func (p *PodmanManager) startContainer(svc ServiceIntent) error {
	image := "docker.io/library/alpine:latest"

	cmd := p.podmanCommand("run", "-d",
		"--name", svc.Name,
		"-p", fmt.Sprintf("%d:80", svc.Port),
		image,
		"sh", "-c", fmt.Sprintf("echo 'Edgelink Service %s running on port %d' && sleep 1000000", svc.Name, svc.Port),
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error al ejecutar podman run: %v", err)
	}

	fmt.Printf("[edgelink-runtime] Contenedor rootless [%s] iniciado y expuesto en el puerto %d\n", svc.Name, svc.Port)
	return nil
}

func (p *PodmanManager) stopContainer(name string) error {
	cmd := p.podmanCommand("stop", name)
	if err := cmd.Run(); err != nil {
		return err
	}
	cmdRm := p.podmanCommand("rm", name)
	_ = cmdRm.Run()
	return nil
}
