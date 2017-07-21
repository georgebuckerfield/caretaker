package caretaker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WhitelistRequest struct {
	Domain    string `json:"domain"`
	IpAddress string `json:"ipaddress"`
}

type WhitelistResponse struct {
	Deadline string
	Status   string
}

func StartServer(interval time.Duration) {
	go backgroundWorker(interval)
	http.HandleFunc("/", processRequest)
	fmt.Printf("[INFO] Server is ready\n")
	http.ListenAndServe(":8000", nil)
}

func backgroundWorker(interval time.Duration) {
	fmt.Printf("[INFO] Starting background worker\n")
	clientset, err := GetClientset()
	if err != nil {
		fmt.Printf("[ERROR] No credentials available\n")
	}
	for range time.Tick(interval) {
		fmt.Printf("[INFO] Starting background cleanup task\n")
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
	var data WhitelistRequest
	var response WhitelistResponse

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&data)

	if err != nil {
		response.Status = fmt.Sprintf("%s", err)
	} else {
		deadline, err := ApplyRequestToCluster(data)
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
