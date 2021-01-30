package main

import (
	"fmt"
	"github.com/insomnimus/libman/cmd"
	"github.com/insomnimus/libman/userdir"
	"io/ioutil"
	"os"
	"strings"
)

const version = "0.9.7"

func showVersion() {
	fmt.Printf("libman version %s\n", version)
}

func showConfigMsg() {
	fmt.Printf("if you don't want to use env variables for your spotify id and secret,\n"+
		"you can edit libman.config file located at %s\n instead.\n\n"+
		"if the config file is missing or corrupted, use `libman reset config` to get a new one.\n"+
		"you can set the LIBMAN_CONFIG_PATH variable to change the default config path.\n", userdir.GetConfigHome() + "/libman")
	os.Exit(0)
}

func resetConfig() {
	fmt.Println("are you sure you want to reset the config file? (y/n)")
	input := cmd.PromptNormal()
	configHome := userdir.GetConfigHome()
	switch strings.ToLower(input) {
	case "y", "yes", "yea":
		err := os.MkdirAll(configHome, 0600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create directory %s: %s\n", configHome, err)
			os.Exit(2)
		}
		err = ioutil.WriteFile(configHome+"/libman.config", []byte(cmd.TemplateConfig), 0600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create template config file at %s: %s\n", configHome+"/libman.config", err)
			fmt.Printf("you can manually create the file and paste this:\n\n%s\n", cmd.TemplateConfig)
			os.Exit(2)
		}
		fmt.Printf("created new empty config file at %s\n", configHome+"/libman.config")
		os.Exit(0)

	default:
		fmt.Println("cancelled")
		os.Exit(0)
	}
}

func resetDB() {
	fmt.Println("this will remove your auth token and you will have to reauthorize. do you want to continue? (y/n)")
	input := cmd.PromptNormal()
	dbPath := userdir.GetDataHome() + "/libman/libman.db"
	switch strings.ToLower(input) {
	case "y", "yes", "yea":
		err := os.Remove(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error removing old db: %s\n", err)
			fmt.Printf("you can however, manually delete the file located at %s (considering it exists)\n", dbPath)
			os.Exit(2)
		}
		fmt.Println("remove successful")
		os.Exit(0)
	default:
		fmt.Println("cancelled")
		os.Exit(0)
	}
}

func main() {
	if len(os.Args) == 1 {
		cmd.StartPlayerSession()
		return
	}
	switch strings.ToLower(os.Args[1]) {
	case "config":
		showConfigMsg()
	case "reset":
		if len(os.Args) == 2 {
			fmt.Fprintln(os.Stderr, `missing argument for reset
		usage:
		libman reset config|db`)
			os.Exit(2)
		}
		switch strings.ToLower(os.Args[2]) {
		case "database", "db", "data":
			resetDB()
		case "config", "configuration":
			resetConfig()
		default:
			fmt.Fprintf(os.Stderr, "unknown option %q for `libman reset`, valid values are `db` or `config`\n", os.Args[2])
			os.Exit(2)
		}

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
