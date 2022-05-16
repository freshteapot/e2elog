package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/freshteapot/e2elog"
	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	coverageOnly := flag.Bool("coverage", false, "Coverage % only")
	statsOnly := flag.Bool("stats", false, "Without endpoint data")
	openapiPath := flag.String("openapi", "./learnalist.yaml", "Path to openapi document")
	logsPath := flag.String("logs", "./logs.ndjson", "Path to logs")
	stripUrlPrefix := flag.String("url-prefix", "/api/v1", "Url Prefix to remove")

	flag.Parse()

	b, _ := ioutil.ReadFile(*openapiPath)
	s, _ := openapi3.NewLoader().LoadFromData(b)

	summary, _ := e2elog.Coverage(s, *logsPath, *stripUrlPrefix)

	if *coverageOnly {
		fmt.Println(summary.Coverage)
		os.Exit(0)
	}

	if *statsOnly {
		stats := e2elog.SummaryStats{
			Total:        summary.Total,
			TotalMatched: summary.TotalMatched,
			Coverage:     summary.Coverage,
		}
		b, _ = json.Marshal(stats)
	}

	if !*statsOnly {
		b, _ = json.Marshal(summary)
	}

	fmt.Println(string(b))
}
