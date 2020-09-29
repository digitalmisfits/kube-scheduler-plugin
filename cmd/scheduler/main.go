package main

import (
	limitAwait "kube-scheduler-plugin/pkg/limit-await"
	"math/rand"
	"os"
	"time"

	"k8s.io/kubernetes/cmd/kube-scheduler/app"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	command := app.NewSchedulerCommand(
		app.WithPlugin(limitAwait.Name, limitAwait.New),
	)

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
