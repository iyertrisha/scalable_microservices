package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/iyertrisha/fleetflow/pkg/auth"
)

func main() {
	sub := "fleetflow-demo"
	if len(os.Args) > 1 {
		sub = os.Args[1]
	}
	tok, err := auth.Issue(sub, 24*time.Hour)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(tok)
}
