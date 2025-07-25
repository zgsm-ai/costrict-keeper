package main

import (
	_ "costrict-host/cmd"
	"costrict-host/cmd/root"
	"log"
	"os"
)

func main() {
	if err := root.RootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}
