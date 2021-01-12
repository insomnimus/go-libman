package cmd

import (
"regexp"
"encoding/json"
"golang.org/x/oauth2/clientcredentials"
"context"
	"fmt"
	"github.com/zmb3/spotify"
	"github.com/boltdb/bolt"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
	"os"
	"os/signal"
)

var(
client *spotify.Client
selectedCache *Cache
caches []*Cache
)

var searchCmd = &cobra.Command{
	Use:   "local",
	Short: "manage your local playlist caches",
	Long: `starts song searching loop`,
	Run: func(cmd *cobra.Command, args []string) {
		startSearchSession()
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

func startSearchSession() {
	signal.Notify(terminator, os.Interrupt)
	dbptr, err:= bolt.Open(dbName, 0600, nil)
	if err!= nil{
		fmt.Fprintf(os.Stderr, "error opening db:\n%s\n", err)
		os.Exit(1)
	}
	db= dbptr
	defer searchCleanup()
	// init spotify client
	config := &clientcredentials.Config{
		ClientID:     os.Getenv("SPOTIFY_ID"),
		ClientSecret: os.Getenv("SPOTIFY_SECRET"),
		TokenURL:     spotify.TokenURL,
	}
	token, err := config.Token(context.Background())
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error initializing spotify client: %s\n", err)
		os.Exit(2)
	}
	tempClient:= spotify.Authenticator{}.NewClient(token)
	client= &tempClient
	fmt.Println("spotify client ready, checking database")
	// check database "cache"
	err= db.View(func(tx *bolt.Tx) error{
		b:= tx.Bucket([]byte("cache"))
		stats:= b.Stats()
		if stats.KeyN > 0{
			c:= b.Cursor()
			for k, v:= c.First(); k!= nil; k, v= c.Next(){
				if v== nil{
					continue
				}
				var cache Cache
				err:= json.Unmarshal(v, &cache)
				if err!=nil{
					return err
				}
				
				caches= append(caches, &cache)
			}
		}
		return nil
	})
	if err!=nil{
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if len(caches)==0{
		fmt.Println("you don't have any cache lists, create one? y/n")
		if !yesOrNo(){
			fmt.Println("exiting")
			os.Exit(2)
		}
		createCache(nil)
		beginSearchLoop()
		return
	}
	chooseCache(nil)
	beginSearchLoop()
}

func chooseCache(args []string) {
	if len(caches) ==0 {
		fmt.Println("you have no caches yet, create one with `new`")
		return
	}
	if len(args) >0{
		name:= concat(args)
		for _, c:= range caches{
			if c.Name== name{
				selectedCache= c
				fmt.Printf("selected cache %s\n", name)
				return
			}
		}
		fmt.Printf("there are no caches with the name %s, create one with `new %s`\n", name, name)
		return
	}
	// interactive
	fmt.Println("choose a local cache")
	for i, c:= range caches{
		fmt.Printf("%d- %s\n", i, c)
	}
	input:= ""
	index:= 0
	for{
		input= prompt()
		if input=="" || input== "-1{
			fmt.Println("returning")
			return
		}
		index, err:= strconv.Atoi(input)
		if err!=nil{
			fmt.Println("invalid input, try again:")
			continue
		}
		if index< 0 || index> len(caches){
			fmt.Printf("invalid input, enter between 0 and %d, enter -1 or blank to return:\n", len(caches)-1)
			continue
		}
		selectedCache= caches[index]
		fmt.Printf("selected cache %s\n", selectedCache.Name)
		return
	}
}

func createCache(args []string) {
	if len(args) >0{
		name:= concat(args)
		for _, c:= range caches{
			if c.Name== name{
				fmt.Printf("there is already a cache with the name %s, select it instead? (y/n)\n")
				if yesOrNo(){
					selectedCache= c
					fmt.Printf("selected cache %s\n", name)
					return
				}
				fmt.Println("not created, returning")
				return
			}
		}
		c:= &Cache{Name: name}
		caches= append(caches, c)
		selectedCache= c
		fmt.Printf("created and selected new cache %s\n", name)
		return
	}
	// interactive
	fmt.Println("creating new cache (exit/cancel to cancel):")
	name:= ""
	for{
		fmt.Println("enter name:")
		name= prompt()
		if name== ""{
			fmt.Println("cancelled")
			return
		}
		if strings.EqualFold(name, "exit") || strings.EqualFold(name, "cancel"){
			fmt.Println("canceled")
			return
		}
		if len(caches)==0{
			c:= &cache{Name: name}
			selectedCache= c
			caches= append(caches, c)
			fmt.Printf("created and selected cache %s\n", c.Name)
			return
		}
		for _, c:= range caches{
			if name== c.Name{
				fmt.Println("there's already a cache with the same name. select it instead? y/n")
				if yesOrNo(){
					selectedCache= c
					fmt.Printf("selected cache %s\n", c.Name)
					return
				}
				fmt.Println("cancelled")
				return
			}
		}
	}
	c:= &Cache{Name: name}
	selectedCache= c
	caches= append(caches, c)
	fmt.Printf("created and selected new cache %s\nreturning\n", name)
}

func searchCleanup() {
	//defer IdentifyPanic()
	if db==nil{
		fmt.Println("warning: db is nil.")
		os.Exit(1)
	}
	err:= db.Update(func(tx *bolt.Tx)error{
		b:= tx.Bucket([]byte("cache"))
		for _, c:= range caches{
			data, err:= json.MarshalIndent(c, "", "\t")
			if err!=nil{
				fmt.Fprintf(os.Stderr, "error doing cleanup: %s\n", err)
				continue
			}
			b.Delete([]byte(c.Name))
			err= b.Put([]byte(c.Name), data)
			if err!=nil{
				fmt.Fprintf(os.Stderr, "error putting %s in db: %s\n", c.Name, err)
				continue
			}
		}
		return nil
	})
	if err !=nil{
		fmt.Fprintf(os.Stderr, "error updating db: %s\n", err)
	}
db.Close()
		os.Exit(0)
}

func searchTrack(s string) {
	if selectedCache== nil{
		fmt.Println("you need to select a cache first\nreturning")
		return
	}
	query:= ""
	song:= ""
	artist:= ""
	if strings.Contains(s, "::"){
		temp:= strings.Split(s, "::")
		if len(temp) != 2{
			song= s
		}else{
			song= temp[0]
			artist= temp[1]
		}
	}else if strings.Contains(s, "-"){
		temp:= strings.Split(s, "-")
		if len(temp) != 2{
			song= s
		}else{
			song= temp[0]
			artist= temp[1]
		}
	}else{
		song= strings.TrimSpace(s)
	}
	song= strings.TrimSpace(song)
	artist= strings.TrimSpace(artist)
	query= "track:" + song
	if artist!= ""{
		query+= " artist:" + artist
	}
	results, err:= client.Search(query, spotify.SearchTypeTrack)
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error searching for %s:\n%s\nreturning\n", query, err)
		return
	}
	if results.Tracks== nil{
		fmt.Printf("no results found for %q\nreturning\n", query)
		return
	}
	tracks:= results.Tracks.Tracks
	if len(tracks)==0{
		fmt.Printf("no results found for %q\nreturning\n", query)
		return
	}
	length:= len(tracks)
	if length > 15{
		tracks= tracks[:15]
		length= 15
	}
	for i, t:= range tracks{
		var artists string
		for _, ar:= range t.Artists{
			artists+= ar.Name + ", "
		}
		fmt.Printf("%d- %s by %s, %s\n",
			i,
			t.Name,
			artists,
			minutes(t.Duration))
	}
	fmt.Printf("add which one to %s? (0-%d), enter -1 or nothing to cancel\n", selectedCache.Name, length-1)
	var input string
	index:= 0
	var track spotify.FullTrack
	for{
		input= prompt()
		if input== "-1" || input== ""{
			fmt.Println("canceled")
			return
		}
		index, err= strconv.Atoi(input)
		if err!=nil{
			fmt.Println("invalid input, enter again:")
			continue
		}
		if index <0 || index >= length{
			fmt.Printf("invalid input, enter between 0 and %d, -1 or blank to cancel:\n", length-1)
			continue
		}
		track= tracks[index]
		break
	}
	selectedCache.Add(track)
	fmt.Println("returning")
}

func deleteCache(args []string) {
	if len(caches) == 0{
		fmt.Println("you do not have any caches, create one with `new`")
		return
	}
	if len(args) != 0{
		name:= concat(args)
		for i, c:= range caches{
			if c.Name== name{
				fmt.Printf("are you sure you want to delete %s from local caches permanently? (y/n)\n", name)
				if yesOrNo(){
					if c.Name== selectedCache.Name{
						selectedCache= nil
					}
					caches= append(caches[:i], caches[i+1:]...)
					fmt.Printf("deleted %s\n", name)
					return
				}
				fmt.Println("cancelled")
				return
			}
		}
		fmt.Printf("there are no caches with the name %s\nreturning\n", name)
		return
	}
	listCaches()
	fmt.Printf("remove which one? (0-%d), -1 or blank to cancel\n", len(caches)-1)
	var input string
	
	for{
		input= prompt()
		if input== "" || input== "-1"{
			fmt.Println("cancelled")
			return
		}
		index, err:= strconv.Atoi(input)
		if err!=nil{
			fmt.Println("invalid input, enter again:")
			continue
		}
		if index <0 || index >= len(caches){
			fmt.Printf("invalid input, enter 0-%d, -1 or blank to return\n", len(caches))
			continue
		}
		// confirm
		fmt.Printf("are you sure you want to remove %s? (y/n)\n", caches[index].Name)
		if yesOrNo(){
			fmt.Printf("removed %s from caches\n", caches[index].Name)
			if caches[index].Name== selectedCache.Name{
				selectedCache=nil
			}
			caches= append(caches[:index], caches[index+1:]...)
			return
		}
		fmt.Println("cancelled")
		return
	}
}

func editCache(args []string) {
	if len(caches)==0{
		fmt.Println("there are no caches, create one with `new`")
		return
	}
	if len(args) !=0{
		name:= concat(args)
		for _, c:= range caches{
			if c.Name== name{
				c.Edit()
				//fmt.Println("returning")
				return
			}
		}
		fmt.Printf("there are no caches with the name %s\nreturning\n", name)
		return
	}
	listCaches()
	fmt.Printf("edit which one? (0-%d), -1 or blank to return\n", len(caches)-1)
	var input string
	//var index int
	for{
		input= prompt()
		if input== "-1" || input== ""{
			fmt.Println("returning")
			return
		}
		index, err:= strconv.Atoi(input)
		if err!=nil{
			fmt.Println("invalid input, enter again:")
			continue
		}
		if index <0 || index>= len(caches){
			fmt.Printf("invalid input, enter 0-%d, blank or -1 to return\n", len(caches)-1)
			continue
		}
		caches[index].Edit()
		//fmt.Println("returning")
		return
	}
}

func listCaches() {
	for i, c:= range caches{
		fmt.Printf("%d- %s\n", i, c.Name)
	}
}

func(c *Cache) Edit() {
	if len(c.Tracks) ==0{
		if c.Name== selectedCache.Name{
			fmt.Printf("%s has no tracks in it, add some with `search`\n", c.Name)
			return
		}
		fmt.Printf("%s has no tracks in it, select and add some? (y/n)\n", c.Name)
		if yesOrNo(){
			selectedCache= c
			fmt.Printf("selected %s\n", c.Name)
			return
		}
		fmt.Println("returning")
		return
	}
	showTracks:= func() {
		for i, t:= range c.Tracks{
		fmt.Printf("%d- %s\n", i, t)
	}
	}
	showTracks()
	fmt.Println("you can remove a song by doing `del <number>`\nto return, enter blank or -1")
	var input string
	var index int
	r:= regexp.MustCompile(`^del[\s]+[0-9]+$`)
	rn:= regexp.MustCompile(`[0-9]+`)
	for{
		input= strings.TrimSpace(strings.ToLower(prompt()))
		if input== "-1" || input== "" || input== "return"{
			fmt.Println("done editing, returning")
			return
		}
		if r.MatchString(input){
			num:= rn.FindString(input)
			// can't fail here
			index, _= strconv.Atoi(num)
			if index <0 || index >= len(c.Tracks){
				fmt.Printf("invalid input, enter between 0-%d, blank or -1 to return\n", len(c.Tracks)-1)
				continue
			}
			fmt.Printf("removed %s from %s\n", c.Tracks[index].Name, c.Name)
			c.Tracks= append(c.Tracks[:index], c.Tracks[index+1:]...)
			if len(c.Tracks) == 0{
				fmt.Printf("%s has no tracks left, returning\n", c.Name)
				return
			}
			showTracks()
			continue
		}
		fmt.Println("invalid input, enter again:")
	}
}

func(c *Cache) List() {
		for i, t:= range c.Tracks{
		fmt.Printf("%d- %s\n", i, t)
	}
}

func showCache(args []string) {
	if len(args) !=0{
		name:= concat(args)
		for _, c:= range caches{
			if strings.EqualFold(name, c.Name){
				c.List()
				return
			}
		}
		fmt.Printf("there is no cache with name %s\n", name)
		return
	}
	if selectedCache==nil{
		fmt.Println("you must select a cache first")
		return
	}
	selectedCache.List()
}