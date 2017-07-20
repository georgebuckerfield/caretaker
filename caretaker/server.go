package caretaker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type WhitelistRequest struct {
	Domain    string `json:"domain"`
	IpAddress string `json:"ipaddress"`
}

func StartServer(interval int) {
	go backgroundWorker(interval)
	http.HandleFunc("/", processRequest)
	fmt.Printf("Server is ready\n")
	http.ListenAndServe(":8000", nil)
}

func backgroundWorker(interval int) {
	clientset, err := GetClientset()
	if err != nil {
		panic("[ERROR] No credentials available")
	}
	for range time.Tick(time.Second * interval) {
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
	var response string
	var data WhitelistRequest

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&data)

	if err != nil {
		response = fmt.Sprintf("%s\n", err)
	} else {
		if err := ApplyRequestToCluster(data); err != nil {
			response = fmt.Sprintf("%s\n", err)
		} else {
			response = "Change successfully applied!\n"
		}
	}

	io.WriteString(w, response)
}
