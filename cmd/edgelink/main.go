package main

import (
	"time"

	"github.com/wzamorab/edgelink/pkg/engine"
)

func main() {
	reconciler := engine.NewReconciler("configs/sample.link.yaml", 5*time.Second)
	reconciler.Start()
}
