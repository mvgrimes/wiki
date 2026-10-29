package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"wiki/cmd"
)

var version = "0.1.3"

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("dotenv load: ", err)
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
