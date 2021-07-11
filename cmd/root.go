package cmd

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/insomnimus/go-libman/userdir"
	"github.com/zmb3/spotify"
	"log"
	"os"
	"path/filepath"
)

const helpText = `
	Libman, a basic spotify controller / library manager
	
	To start a session, run the application with no arguments.
	
	subcommands:
	#config: get details about configuration
	#reset: reset config or db
	#version: display the libman version
	#help: show this message
	
	Note:
	libman needs to store the caches and the access token somewhere. by default, all the data is stored
	under the users default data path (~/.local for linux and %APPDATA% for windows).
	you can however, set the "LIBMAN_DB_PATH" environment variable to target somewhere else (windows users should use forward slashes as well)
	
	libman expects these env variables to be set:
	$SPOTIFY_ID
	$SPOTIFY_SECRET
	
	you will have to register an application at: https://developer.spotify.com/my-applications/
	- Use "http://localhost:8080/callback" as the redirect URI
	
	If you don't want to use env variables, run the "libman config" command for more options.
	`

var (
	LibmanConfig = userdir.LibmanConfig()
	db           *bolt.DB
)

func init() {
	if _, err := os.Stat(LibmanConfig.DBPath); os.IsNotExist(err) {
		fmt.Println("no database detected, initializing one")
		initDB()
		fmt.Println("db created")
	}
}

type Playlist struct {
	Name        string  `json:"name"`
	ID          string  `json:"id"`
	Description string  `json:"description"`
	Tracks      []Track `json:"tracks"`
	removeCache []Track
	addCache    []Track
}

type Track struct {
	Name     string   `json:"name"`
	ID       string   `json:"id"`
	Artists  []string `json:"artists"`
	URI      spotify.URI
	Duration string
}

func (t Track) String() string {
	artists := ""
	for _, art := range t.Artists {
		artists += ", " + art
	}
	if artists != "" {
		artists = artists[1:]
	}

	msg := fmt.Sprintf("%s by %s", t.Name, artists)
	if t.Duration != "" {
		msg += " | " + t.Duration
	}
	return msg
}

func initDB() {
	dbDir := filepath.Dir(LibmanConfig.DBPath)
	err := os.MkdirAll(dbDir, 0600)
	if err != nil {
		log.Fatal(err)
	}
	db, err := bolt.Open(LibmanConfig.DBPath, 0600, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating database:\n%s\n", err)
		os.Exit(1)
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte("token"))
		if err != nil {
			return fmt.Errorf("create bucket: token: %s", err)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "db error:\n%s\n", err)
		os.Exit(1)
	}
}

func ShowHelp() {
	fmt.Println(helpText)
}
