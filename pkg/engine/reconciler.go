package engine

import (
	"fmt"
	"os"
	"time"

	"github.com/wzamorab/edgelink/pkg/crypto"
	"github.com/wzamorab/edgelink/pkg/firewall"
	"github.com/wzamorab/edgelink/pkg/manifest"
	"github.com/wzamorab/edgelink/pkg/overlay"
	"github.com/wzamorab/edgelink/pkg/runtime"
)

type Reconciler struct {
	ManifestPath    string
	Interval        time.Duration
	OverlayManager  *overlay.OverlayManager
	FirewallManager *firewall.FirewallManager
	CryptoValidator *crypto.Validator
	TriggerManager  *TriggerManager
	PodmanManager   *runtime.PodmanManager
}

func NewReconciler(path string, interval time.Duration) *Reconciler {
	validator := crypto.NewValidator()
	_ = validator.RegisterFederationKey("fed.personal", "3b612713d2899453762635a960133c94f54e17812f8e65842817cc9dfb7d60e7")

	return &Reconciler{
		ManifestPath:    path,
		Interval:        interval,
		OverlayManager:  overlay.NewOverlayManager("edgenet0"),
		FirewallManager: firewall.NewFirewallManager(),
		CryptoValidator: validator,
		TriggerManager:  NewTriggerManager(),
		PodmanManager:   runtime.NewPodmanManager(),
	}
}

func (r *Reconciler) Start() {
	ticker := time.NewTicker(r.Interval)
	defer ticker.Stop()

	fmt.Println("[edgelink] Motor de reconciliación declarativa con Podman rootless iniciado...")

	for range ticker.C {
		rawdata, err := os.ReadFile(r.ManifestPath)
		if err != nil {
			continue
		}

		cfg, err := manifest.LoadManifest(r.ManifestPath)
		if err != nil {
			fmt.Printf("[Error Parseo Manifiesto] %v\n", err)
			continue
		}

		r.Converge(cfg, rawdata)
	}
}

func (r *Reconciler) Converge(cfg *manifest.IntentManifest, rawdata []byte) {
	// 1. Evaluar disparadores ambientales
	if dynamicScope := r.TriggerManager.EvaluateTriggers(cfg.ScopeContext.Triggers); dynamicScope != "" {
		cfg.ScopeContext.ActiveScope = dynamicScope
	}

	fmt.Printf("[Converge] Nodo: %s | Federación: %s | Ámbito Activo: %s\n",
		cfg.Metadata.NodeID,
		cfg.Metadata.Federation,
		cfg.ScopeContext.ActiveScope,
	)

	// 2. Reconciliar red overlay TUN
	err := r.OverlayManager.EnsureState(cfg.Network.OverlaySubnet)
	if err != nil {
		fmt.Printf("[Error Overlay] %v\n", err)
		return
	}

	// 3. Reconciliar políticas de firewall según el ámbito
	err = r.FirewallManager.ApplyScope(cfg.ScopeContext.ActiveScope)
	if err != nil {
		fmt.Printf("[Error Firewall] %v\n", err)
		return
	}

	// 4. Reconciliar cargas de trabajo con Podman rootless
	for _, svc := range cfg.Services {
		runtimeSvc := runtime.ServiceIntent{
			Name:   svc.Name,
			Status: svc.Status,
			Port:   svc.Port,
		}
		err := r.PodmanManager.ReconcileService(runtimeSvc, cfg.Network.OverlaySubnet)
		if err != nil {
			fmt.Printf("[Error Runtime Podman] %v\n", err)
		}
	}
}
