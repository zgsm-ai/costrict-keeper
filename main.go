package main

import (
	_ "costrict-keeper/cmd"
	"costrict-keeper/cmd/root"
	"log"
	"os"
)

func main() {
	if err := root.RootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}
