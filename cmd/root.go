package cmd

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/insomnimus/libman/userdir"
	"github.com/spf13/cobra"
	"github.com/zmb3/spotify"
	"os"
	"strings"
)

const helpText = `usage:
	libman <subcommand>
	
	subcommands are:
	#player | to control playback and do simple library management
	#live | no playback, more control over library management
	#local | doesn't require authentication, modify local caches for later syncing
	
	you can set the LIBMAN_DEFAULT_CMD environment variable to one of the commands (player, live, local)
	to add the default behaviour to the bare "libman" command
	
	Note:
	libman needs to store the caches and the access token somewhere. by default, all the data is stored
	under the users default data path (~/.local for linux and %APPDATA% for windows).
	you can however, set the "LIBMAN_DB_PATH" environment variable to target somewhere else (windows users should use forward slashes as well)
	
	libman expects these env variables to be set:
	$SPOTIFY_ID
	$SPOTIFY_SECRET
	
	you will have to register an application at: https://developer.spotify.com/my-applications/
	- Use "http://localhost:8080/callback" as the redirect URI
	
	`

var COMMAND = ""
var (
	dbPath string = userdir.GetDataHome() + "/libman"
	dbName string = dbPath + "/libman.db"
	db     *bolt.DB
)

var rootCmd = &cobra.Command{
	Use:   "libman",
	Short: "a spotify library manager",
	Long:  helpText,

	Run: func(cmd *cobra.Command, args []string) {
		libmanDefault := os.Getenv("LIBMAN_DEFAULT_CMD")
		switch strings.ToLower(libmanDefault) {
		case "player":
			COMMAND = "player"
			startPlayerSession()
		case "live":
			COMMAND = "live"
			startLiveSession()
		case "local":
			COMMAND = "local"
			startSearchSession()
		case "":
			fmt.Println(helpText)
			os.Exit(0)
		default:
			fmt.Println(helpText)
			fmt.Printf("note: your LIBMAN_DEFAULT_CMD variable is set to %q, which is not valid.\nvalid values are 'player', 'local' and 'live'\n", libmanDefault)
			os.Exit(0)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		if fileErr := os.MkdirAll(dbPath, 0764); fileErr != nil {
			fmt.Fprintf(os.Stderr, "failed to create a directory for database in %s\n", fileErr)
			os.Exit(1)
		}
		fmt.Printf("created a directory in %s\n", dbPath)
	}
	if _, err := os.Stat(dbName); os.IsNotExist(err) {
		fmt.Println("no database detected, initializing one")
		initDB()
		fmt.Println("db created")
	}

	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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
	Name    string   `json:"name"`
	ID      string   `json:"id"`
	Artists []string `json:"artists"`
	URI     spotify.URI
}

func (t Track) String() string {
	artists := ""
	for _, art := range t.Artists {
		artists += ", " + art
	}
	if artists != "" {
		artists = artists[1:]
	}
	return fmt.Sprintf("%s by %s", t.Name, artists)

}

type Cache struct {
	Name   string  `json:"name"`
	Tracks []Track `json:"tracks"`
}

func (c *Cache) Add(song spotify.FullTrack) {
	var artists []string
	for _, ar := range song.Artists {
		artists = append(artists, ar.Name)
	}
	if len(c.Tracks) == 0 {
		c.Tracks = append(c.Tracks, Track{
			Name:    song.Name,
			Artists: artists,
			ID:      string(song.ID),
		})
		fmt.Printf("added %s to %s\n", song.Name, c.Name)
		return
	}
	for _, t := range c.Tracks {
		if string(song.ID) == t.ID {
			fmt.Printf("%s already has the track %s, not added.\n", c.Name, song.Name)
			return
		}
	}
	c.Tracks = append(c.Tracks, Track{
		Name:    song.Name,
		Artists: artists,
		ID:      string(song.ID),
	})
	fmt.Printf("added %s to %s\n", song.Name, c.Name)
}

func (c *Cache) String() string {
	return fmt.Sprintf("%s (%d tracks)", c.Name, len(c.Tracks))
}

func initDB() {
	db, err := bolt.Open(dbName, 0600, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating database:\n%s\n", err)
		os.Exit(1)
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte("playlists"))
		if err != nil {
			return fmt.Errorf("create bucket playlists: %s", err)
		}
		_, err = tx.CreateBucket([]byte("cache"))
		if err != nil {
			return fmt.Errorf("create bucket cache: %s", err)
		}
		_, err = tx.CreateBucket([]byte("token"))
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
