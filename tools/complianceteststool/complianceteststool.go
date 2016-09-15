package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/readium/readium-lcp-server/logging"
)

func main() {
	logFilePath := flag.String("logfilepath", "", "path to .log file")
	flag.Parse()

	var input string
	var result string

	for {
		err := logging.Init(*logFilePath, true)
		if err != nil {
			panic(err)
		}

		fmt.Println("Enter the number of tests, 'q' for quit the tool")
		fmt.Scanln(&input)

		if strings.EqualFold(input, "q") {
			os.Exit(0)
		}

		for {
			fmt.Println("Enter the result of test ('f' if test failed, 's' if test has success)")
			fmt.Scanln(&result)

			if strings.EqualFold(result, "f") {
				logging.WriteToFile("error", 0, input)
				break
			}

			if strings.EqualFold(result, "s") {
				logging.WriteToFile("success", 0, input)
				break
			}
		}
	}
}
