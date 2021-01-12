package cmd

import (
"github.com/boltdb/bolt"
"os"
"libman/userdir"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/zmb3/spotify"
)

var(
dbPath string= userdir.GetDataHome() + "/libman"
dbName string= dbPath + "/libman.db"
db *bolt.DB
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "libman",
	Short: "a spotify library manager",
	Long: `usage:
	libman <subcommand> [flags]
	
	subcommands are:
	#search
	#live`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	if _, err:= os.Stat(dbPath); os.IsNotExist(err){
		if fileErr := os.Mkdir(dbPath, 0764); fileErr!=nil{
			fmt.Fprintf(os.Stderr, "failed to create a directory for database in %s\n", fileErr)
			os.Exit(1)
		}
		fmt.Printf("created a directory in %s\n", dbPath)
	}
	if _, err:= os.Stat(dbName); os.IsNotExist(err){
		fmt.Println("no database detected, initializing one")
		initDB()
		fmt.Println("db created")
	}
	
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.libman.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

type Playlist struct{
	Name string `json:"name"`
	ID string `json:"id"`
	Description string `json:"description"`
	Tracks []Track `json:"tracks"`
	removeCache []Track
	addCache []Track
}

type Track struct{
	Name string `json:"name"`
	ID string `json:"id"`
	Artists []string `json:"artists"`
}

func(t Track) String() string{
	artists:= ""
	for _, art:= range t.Artists{
		artists+= ", " + art
	}
	if artists!= ""{
		artists= artists[1:]
	}
	return fmt.Sprintf("%s by %s", t.Name, artists)
	
}

type Cache struct{
	Name string `json:"name"`
	Tracks []Track `json:"tracks"`
}

func(c *Cache) Add(song spotify.FullTrack) {
	var artists []string
	for _, ar:= range song.Artists{
		artists= append(artists, ar.Name)
	}
	if len(c.Tracks) ==0 {
		c.Tracks= append(c.Tracks, Track{
			Name: song.Name,
			Artists: artists,
			ID: string(song.ID),
		})
		fmt.Printf("added %s to %s\n", song.Name, c.Name)
		return
	}
	for _, t:= range c.Tracks{
		if string(song.ID) == t.ID{
			fmt.Printf("%s already has the track %s, not added.\n", c.Name, song.Name)
			return
		}
	}
	c.Tracks= append(c.Tracks, Track{
		Name: song.Name,
		Artists: artists,
		ID: string(song.ID),
	})
	fmt.Printf("added %s to %s\n", song.Name, c.Name)
}

func(c *Cache) String() string{
	return fmt.Sprintf("%s (%d tracks)", c.Name, len(c.Tracks))
}

func initDB() {
	db, err:= bolt.Open(dbName, 0600, nil)
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error creating database:\n%s\n", err)
		os.Exit(1)
	}
	defer db.Close()
	err= db.Update(func(tx *bolt.Tx) error{
		_, err:= tx.CreateBucket([]byte("playlists"))
		if err!=nil{
			return fmt.Errorf("create bucket playlists: %s", err)
		}
		_, err= tx.CreateBucket([]byte("cache"))
		if err!= nil{
			return fmt.Errorf("create bucket cache: %s", err)
		}
		_, err= tx.CreateBucket([]byte("token"))
		if err!=nil{
			return fmt.Errorf("create bucket: token: %s", err)
		}
		return nil
	})
	if err!=nil{
		fmt.Fprintf(os.Stderr, "db error:\n%s\n", err)
		os.Exit(1)
	}
}