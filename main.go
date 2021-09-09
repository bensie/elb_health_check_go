package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

// Hostnames contain the list of hosts that are being checked agains.
var Hostnames = strings.Split(os.Getenv("HEALTH_CHECK_HOSTNAMES"), ",")

// The port the host is listening on, usually 80
var HostPort = os.Getenv("HEALTH_CHECK_HOST_PORT")

// HTTPResponse contains the http response data to the specified host
type HTTPResponse struct {
	hostname string
	response *http.Response
	err      error
	check    string
}

// CheckResponse contains the HTTP status response from the specified host
type CheckResponse struct {
	Status int    `json:"status"`
	Check  string `json:"check"`
}

func main() {
	port := ""
	flag.StringVar(&port, "port", "9292", "the port to listen on")

	flag.Parse()

	fmt.Fprintln(os.Stderr, "starting listener on port", port)
	http.HandleFunc("/", mainHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	allowedToFailQueryParam := r.URL.Query().Get("allowed_to_fail")
	mustSucceedQueryParam := r.URL.Query().Get("must_succeed")

	hostnamesAllowedToFail := strings.Split(allowedToFailQueryParam, ",")
	hostnamesMustSucceed := strings.Split(mustSucceedQueryParam, ",")

	ch := make(chan *HTTPResponse, len(Hostnames))
	for _, hostname := range Hostnames {
		go makeRequest(hostname, ch)
	}

	// Block until we get responses for all hostnames
	responses := []*HTTPResponse{}
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

	filteredResponses := []*HTTPResponse{}
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

func makeRequest(hostname string, ch chan<- *HTTPResponse) {
	client := &http.Client{}

	if HostPort == "" {
		HostPort = "80"
	}

	check_url := fmt.Sprintf("http://0.0.0.0:%s/health_check", HostPort)
	req, _ := http.NewRequest("HEAD", check_url, nil)
	req.Host = hostname
	req.Header.Add("X-Forwarded-Proto", "https")

	resp, err := client.Do(req)

	check := "failure"
	if resp != nil {
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode <= 209 {
			check = "success"
		}
	}

	ch <- &HTTPResponse{hostname, resp, err, check}
}

func contains(stringSlice []string, searchString string) bool {
	for _, value := range stringSlice {
		if value == searchString {
			return true
		}
	}
	return false
}
