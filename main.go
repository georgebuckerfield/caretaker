package main

import (
	"github.com/georgebuckerfield/caretaker/caretaker"
	"os"
)

const (
	envConfigInterval     = "BACKGROUND_WORKER_INTERVAL"
	defaultConfigInterval = 60
)

func main() {

	// Interval sets the frequency of the background worker:
	var interval int
	if interval := os.Getenv(envConfigInterval); interval == "" {
		interval = defaultConfigInterval
	}

	caretaker.StartServer(interval)
}
