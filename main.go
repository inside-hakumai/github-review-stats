package main

import (
	"github.com/inside-hakumai/github-review-stats/cmd"
	"log"
)

func main() {
	err := cmd.Exec()
	if err != nil {
		log.Fatalf("%+v\n", err)
	}
}
