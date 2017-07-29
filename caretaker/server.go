package caretaker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type contextKey string

type WhitelistRequest struct {
	Domain    string `json:"domain"`
	IpAddress string `json:"ipaddress"`
}

type WhitelistResponse struct {
	Deadline string
	Status   string
}

const (
	requestTimeKey = "requestTime"
)

func StartServer(interval time.Duration) {
	go backgroundWorker(interval)
	http.HandleFunc("/", processRequest)
	fmt.Printf("Server is ready\n")
	http.ListenAndServe(":8000", nil)
}

func backgroundWorker(interval time.Duration) {
	fmt.Printf("Starting background worker\n")
	clientset, err := GetClientset()
	if err != nil {
		fmt.Printf("No credentials available\n")
	}
	for range time.Tick(interval) {
		fmt.Printf("Starting background cleanup task\n")
		services := GetServiceList(clientset)
		for _, s := range services.Items {
			if IsAutoManaged(&s) {
				err := IterateAnnotations(&s, clientset)
				if err != nil {
					fmt.Printf("%s\n", err)
				}
			}
		}
	}
}

func processRequest(w http.ResponseWriter, r *http.Request) {
	val := time.Now()
	key := contextKey(requestTimeKey)
	ctx := context.WithValue(context.Background(), key, val)

	var (
		data     WhitelistRequest
		response WhitelistResponse
	)

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&data)

	if err != nil {
		response.Status = fmt.Sprintf("%s", err)
	} else {
		deadline, err := ApplyRequestToCluster(ctx, data)
		if err != nil {
			response.Status = fmt.Sprintf("%s", err)
		} else {
			response.Status = fmt.Sprintf("IP successfully whitelisted until: %s", deadline)
			response.Deadline = deadline
		}
	}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}
