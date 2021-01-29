package main

import (
	"fmt"
	"github.com/insomnimus/libman/cmd"
	"os"
	"strings"
)

const version = "0.9.7"

func showVersion() {
	fmt.Printf("libman version %s\n", version)
}

func main() {
	if len(os.Args) == 1 {
		cmd.StartPlayerSession()
		return
	}
	switch strings.ToLower(os.Args[1]) {
	case "help", "-h", "--help":
		cmd.ShowHelp()
		return
	case "version", "-v", "--version":
		showVersion()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown option/command %q\n", os.Args[1])
		return
	}
}
