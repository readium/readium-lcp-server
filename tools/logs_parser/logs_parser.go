package main

import (
	"flag"
	"fmt"

	"github.com/readium/readium-lcp-server/logging"
)

func main() {
	logFilePath := flag.String("logfilepath", "", "path to .log file")
	parsedFilePath := flag.String("parsedfilepath", "", "path to .log file")

	fmt.Println(parsedFilePath)

	flag.Parse()

	err := logging.Init(*logFilePath, true)
	if err != nil {
		panic(err)
	}

	fmt.Println("Parsing log file...")
	logs, err := logging.ReadLogs(*logFilePath)
	summary, err := logging.CountTotal(logs)

	if err != nil {
		panic(err)
	}

	fmt.Println(logs)
	fmt.Println(summary)
}
