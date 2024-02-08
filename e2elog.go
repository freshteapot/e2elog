package e2elog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gorilla/mux"
	"github.com/thoas/go-funk"
)

type SummaryStats struct {
	Total        int     `json:"total"`
	TotalMatched int     `json:"total_matched"`
	Coverage     float64 `json:"coverage"`
}

type Summary struct {
	Total        int        `json:"total"`
	TotalMatched int        `json:"total_matched"`
	Endpoints    []Endpoint `json:"endpoints"`
	Coverage     float64    `json:"coverage"`
}

type HTTPLog struct {
	Method     string `json:"method"`
	URL        string `json:"url"`
	StatusCode int    `json:"status_code"`
}

type Endpoint struct {
	Path        string `json:"path"`
	Method      string `json:"method"`
	StatusCode  string `json:"status_code"`
	OperationID string `json:"operation_id"`
	Touched     bool   `json:"touched"`
}

// Coverage based on openapi document and a list of logs,
// Link the logs to the openapi paths to get insight into
// how many endpoints have been touched
func Coverage(s *openapi3.T, pathToLogs string, stripUrlPrefix string) (Summary, error) {
	endpoints := getPaths(s)
	router := createRouter(endpoints)

	file, err := os.Open(pathToLogs)
	if err != nil {
		return Summary{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var httpLog HTTPLog
		json.Unmarshal(scanner.Bytes(), &httpLog)

		httpLog.URL = strings.TrimPrefix(httpLog.URL, stripUrlPrefix)

		response := lookUp(router, httpLog)
		if response.StatusCode != http.StatusOK {
			continue
		}

		endpointIndex := funk.IndexOf(endpoints, func(endpoint Endpoint) bool {
			if endpoint.OperationID == response.Header.Get("operation-id") {
				if endpoint.Method == httpLog.Method {
					if endpoint.StatusCode == strconv.Itoa(httpLog.StatusCode) {
						return true
					}
				}
			}
			return false
		})

		if endpointIndex == -1 {
			continue
		}

		endpoint := endpoints[endpointIndex]
		endpoint.Touched = true
		endpoints[endpointIndex] = endpoint
	}

	if err := scanner.Err(); err != nil {
		return Summary{}, err
	}

	total := len(endpoints)
	totalFound := len(funk.Filter(endpoints, func(endpoint Endpoint) bool {
		return endpoint.Touched
	}).([]Endpoint))

	return Summary{
		Total:        total,
		TotalMatched: totalFound,
		Endpoints:    endpoints,
		Coverage:     float64(total) / float64(totalFound),
	}, nil
}

// getPaths
func getPaths(s *openapi3.T) []Endpoint {
	var endpoints []Endpoint
	for uri, path := range s.Paths.Map() {
		fmt.Println(uri)
		for method, operation := range path.Operations() {

			for statusCode, _ := range operation.Responses.Map() {
				endpoint := Endpoint{
					Path:        uri,
					Method:      method,
					StatusCode:  statusCode,
					OperationID: operation.OperationID,
				}

				endpoints = append(endpoints, endpoint)
			}
		}
	}
	return endpoints
}

func createRouter(endpoints []Endpoint) *mux.Router {
	// Get a list of unique endpoints
	// Make unique based on path + operationID + method
	unique := funk.Uniq(funk.Map(endpoints, func(endpoint Endpoint) Endpoint {
		return Endpoint{
			Path:        endpoint.Path,
			OperationID: endpoint.OperationID,
			Method:      endpoint.Method,
		}
	}).([]Endpoint)).([]Endpoint)

	notFound := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	r := mux.NewRouter()
	r.MethodNotAllowedHandler = notFound
	r.NotFoundHandler = notFound

	for _, endpoint := range unique {
		// This was required or it just kept inherting the latest endpoint
		operationID := endpoint.OperationID
		r.Handle(endpoint.Path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("operation-id", operationID)
			w.WriteHeader(http.StatusOK)
		})).Methods(endpoint.Method)
	}

	return r
}

func lookUp(r *mux.Router, request HTTPLog) *http.Response {
	rec := httptest.NewRecorder()
	req := newRequest(request.Method, request.URL)
	r.ServeHTTP(rec, req)
	return rec.Result()
}

// From mux repo
// https://github.com/gorilla/mux/blob/master/mux_test.go
func newRequest(method, url string) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	// extract the escaped original host+path from url
	// http://localhost/path/here?v=1#frag -> //localhost/path/here
	opaque := ""
	if i := len(req.URL.Scheme); i > 0 {
		opaque = url[i+1:]
	}

	if i := strings.LastIndex(opaque, "?"); i > -1 {
		opaque = opaque[:i]
	}
	if i := strings.LastIndex(opaque, "#"); i > -1 {
		opaque = opaque[:i]
	}

	// Escaped host+path workaround as detailed in https://golang.org/pkg/net/url/#URL
	// for < 1.5 client side workaround
	req.URL.Opaque = opaque

	// Simulate writing to wire
	var buff bytes.Buffer
	req.Write(&buff)
	ioreader := bufio.NewReader(&buff)

	// Parse request off of 'wire'
	req, err = http.ReadRequest(ioreader)
	if err != nil {
		panic(err)
	}
	return req
}
