package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

var Hostnames = strings.Split(os.Getenv("HEALTH_CHECK_HOSTNAMES"), ",")

type HttpResponse struct {
	hostname string
	response *http.Response
	err      error
	check    string
}

type CheckResponse struct {
	Status int    `json:"status"`
	Check  string `json:"check"`
}

func main() {
	http.HandleFunc("/", mainHandler)
	log.Fatal(http.ListenAndServe(":9292", nil))
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	allowedToFailQueryParam := r.URL.Query().Get("allowed_to_fail")
	mustSucceedQueryParam := r.URL.Query().Get("must_succeed")

	hostnamesAllowedToFail := strings.Split(allowedToFailQueryParam, ",")
	hostnamesMustSucceed := strings.Split(mustSucceedQueryParam, ",")

	ch := make(chan *HttpResponse, len(Hostnames))
	for _, hostname := range Hostnames {
		go makeRequest(hostname, ch)
	}

	// Block until we get responses for all hostnames
	responses := []*HttpResponse{}
	for range Hostnames {
		responses = append(responses, <-ch)
	}

	httpResponseData := make(map[string]CheckResponse)
	for _, r := range responses {
		var code int
		if r.response != nil {
			code = r.response.StatusCode
		}
		httpResponseData[r.hostname] = CheckResponse{code, r.check}
	}

	filteredResponses := []*HttpResponse{}
	if len(hostnamesAllowedToFail) > 0 {
		for _, r := range responses {
			if !contains(hostnamesAllowedToFail, r.hostname) {
				filteredResponses = append(filteredResponses, r)
			}
		}
	} else if len(hostnamesMustSucceed) > 0 {
		for _, r := range responses {
			if contains(hostnamesMustSucceed, r.hostname) {
				filteredResponses = append(filteredResponses, r)
			}
		}
	} else {
		for _, r := range responses {
			filteredResponses = append(filteredResponses, r)
		}
	}

	failedAnywhere := false
	for _, fr := range filteredResponses {
		if fr.check == "failure" {
			failedAnywhere = true
		}
	}

	if failedAnywhere {
		w.WriteHeader(http.StatusInternalServerError)
	}

	json, err := json.Marshal(httpResponseData)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Fprintf(w, string(json))
}

func makeRequest(hostname string, ch chan<- *HttpResponse) {
	client := &http.Client{}

	req, _ := http.NewRequest("HEAD", "http://0.0.0.0:80/health_check", nil)
	req.Header.Add("Host", hostname)
	req.Header.Add("X-Forwarded-Proto", "https")

	resp, err := client.Do(req)

	check := "failure"
	if resp != nil {
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode <= 209 {
			check = "success"
		}
	}

	ch <- &HttpResponse{hostname, resp, err, check}
}

func contains(stringSlice []string, searchString string) bool {
	for _, value := range stringSlice {
		if value == searchString {
			return true
		}
	}
	return false
}
