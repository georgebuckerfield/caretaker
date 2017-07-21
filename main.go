package main

import (
	"os"
	"strconv"
	"time"

	"github.com/georgebuckerfield/caretaker/caretaker"
)

const (
	envConfigInterval     = "BACKGROUND_WORKER_INTERVAL"
	defaultConfigInterval = 60
)

func main() {

	// Interval sets the frequency of the background worker:
	var interval time.Duration

	envInterval, err := strconv.Atoi(os.Getenv(envConfigInterval))
	if err != nil {
		interval = time.Duration(defaultConfigInterval) * time.Second
	} else {
		interval = time.Duration(envInterval) * time.Second
	}
	caretaker.StartServer(interval)
}
