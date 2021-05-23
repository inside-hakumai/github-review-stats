package main

import (
	"github.com/inside-hakumai/github-review-stats/cmd"
	"log"
	"os"
)

func main() {
	flagValues, isValid := cmd.ParseCommandLineArguments()
	if !isValid {
		os.Exit(1)
	}

	err := cmd.Exec(*flagValues)
	if err != nil {
		log.Fatalf("%+v\n", err)
	}
}
