package cmd

import (
"os/signal"
"strconv"
"golang.org/x/oauth2"
"github.com/boltdb/bolt"
"os"
"encoding/json"
"log"
"net/http"
	"fmt"
"strings"

"github.com/zmb3/spotify"
	"github.com/spf13/cobra"
)

const redirectURI = "http://localhost:8080/callback"

var (
user *spotify.PrivateUser
	// all the permissions
	auth  = spotify.NewAuthenticator(redirectURI,
	spotify.ScopeImageUpload,
spotify.ScopePlaylistReadPrivate,
spotify.ScopePlaylistModifyPublic,
spotify.ScopePlaylistModifyPrivate,
spotify.ScopePlaylistReadCollaborative,
spotify.ScopeUserFollowModify,
spotify.ScopeUserFollowRead,
spotify.ScopeUserLibraryModify,
spotify.ScopeUserLibraryRead,
spotify.ScopeUserReadPrivate,
spotify.ScopeUserReadEmail,
spotify.ScopeUserReadCurrentlyPlaying,
spotify.ScopeUserReadPlaybackState,
spotify.ScopeUserModifyPlaybackState,
spotify.ScopeUserReadRecentlyPlayed,
spotify.ScopeUserTopRead,
spotify.ScopeStreaming,
)
	ch    = make(chan *spotify.Client)
	state = "abc123"
)

var(
simplePlaylists []spotify.SimplePlaylist
//fullPlaylists []spotify.FullPlaylist
//selectedFull *spotify.FullPlaylist
selectedSimple *spotify.SimplePlaylist
)

var liveCmd = &cobra.Command{
	Use:   "live",
	Short: "starts an authenticated spotify session",
	Long: `for editing of the personal spotify playlists`,
	Run: func(cmd *cobra.Command, args []string) {
		startLiveSession()
	},
}

func startLiveSession() {
	defer IdentifyPanic()
	defer liveCleanup()
	signal.Notify(terminator, os.Interrupt)
	
	checkToken()
	// load caches
	err:= db.View(func(tx *bolt.Tx) error{
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
	for{
		parseLiveCommand(prompt())
	}
}

func init() {
	rootCmd.AddCommand(liveCmd)
}

func checkToken() {
	if db==nil{
		dbptr, err:= bolt.Open(dbName, 0600, nil)
		if err!=nil{
			fmt.Fprintf(os.Stderr, "error opening db: %s\n", err)
			os.Exit(1)
		}
		db= dbptr
	}
	var data []byte
	db.View(func(tx *bolt.Tx) error{
		b:= tx.Bucket([]byte("token"))
		data=b.Get([]byte("token"))
		return nil
	})
	if data== nil{
		authorize()
		return
	}
	var token oauth2.Token
	err:= json.Unmarshal(data, &token)
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error unmarshaling token: %s\n", err)
		os.Exit(1)
	}
	initClient(&token)
}

func authorize() {
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//log.Println("Got request for:", r.URL.String())
	})
	go http.ListenAndServe(":8080", nil)

	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)
	// wait for auth to complete
	clt:= <-ch

	// use the client to make calls that require authorization
	usr, err := clt.CurrentUser()
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error authorizing: %s\n", err)
		os.Exit(1)
	}
	client= clt
	user= usr
	token, err:= client.Token()
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error saving access token: %s\n", err)
		os.Exit(1)
	}
	data, err:= json.Marshal(token)
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error marshaling token: %s\n", err)
		os.Exit(1)
	}
	err= db.Update(func(tx *bolt.Tx)error{
		b:= tx.Bucket([]byte("token"))
		err:= b.Put([]byte("token"), data)
		return err
	})
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error saving access token to db: %s\n", err)
		os.Exit(1)
	}
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}
	// use the token to get an authenticated client
	client := auth.NewClient(tok)
	fmt.Fprintf(w, "Login Completed!")
	ch <- &client
}

func initClient(token *oauth2.Token) {
	clt:= auth.NewClient(token)
	client= &clt
	usr, err:= client.CurrentUser()
	if err!=nil{
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	user= usr
	fmt.Printf("welcome %s\n", user.DisplayName)
}

func choosePlaylist(args []string) {
	page, err:= client.CurrentUsersPlaylists()
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error fetching users playlists: %s\n", err)
		os.Exit(1)
	}
	if len(page.Playlists)==0{
		fmt.Println("you don't seem to have any playlist, create one with `new`")
		return
	}
	simplePlaylists= page.Playlists
	if len(args) != 0{
		name:= concat(args)
		for _, pl:= range simplePlaylists{
			if strings.EqualFold(pl.Name, name){
				fmt.Printf("selected playlist %s\n", name)
				selectedSimple= &pl
				return
			}
		}
		fmt.Printf("there are no playlists with the name %s, use `new` to create one\n", name)
		return
	}
	for i, pl:= range simplePlaylists{
		fmt.Printf("%d- %s\n", i, pl.Name)
	}
	fmt.Printf("choose playlist (0-%d), -1 or blank to cancel\n", len(simplePlaylists)-1)
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
		if index<0 || index>= len(simplePlaylists){
			fmt.Printf("invalid input, enter 0-%d, -1 or blank to cancel\n", len(simplePlaylists)-1)
			continue
		}
		selectedSimple= &simplePlaylists[index]
		fmt.Printf("selected %s\n", selectedSimple.Name)
		return
	}
}

func editSelectedPlaylist() {
	if selectedSimple== nil{
		fmt.Println("you need to select a playlist first, use `select`")
		return
	}
	page, err:= client.GetPlaylistTracks(selectedSimple.ID)
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error fetching the list of tracks in %s:\n%s\n", selectedSimple.Name, err)
		os.Exit(1)
	}
	draft:= &Playlist{Name: selectedSimple.Name, ID: string(selectedSimple.ID)}
	currentPage:= 0
	if len(page.Tracks)>0 {
		for _, tr:= range page.Tracks{
			track:= Track{
				Name: tr.Track.Name,
				ID: string(tr.Track.ID),
			}
			for _, art:= range tr.Track.Artists{
				track.Artists= append(track.Artists, art.Name)
			}
			draft.Tracks= append(draft.Tracks, track)
		}
	}
	draft.Display(currentPage)
	changePage:= func(step int) {
		if currentPage+ step <0 {
			fmt.Println("you can't go back page 1")
			return
		}
		if (currentPage + step -1) * 40 >= len(draft.Tracks){
			fmt.Println("this is already the last page")
			return
		}
		currentPage+= step
		draft.Display(currentPage)
	}
	var input string
	var fields []string
	for{
		input= prompt()
		if input== "-1" || input== ""{
			if len(draft.addCache) !=0 || len(draft.removeCache)!=0 {
				fmt.Println("you have unsaved changes, commit them? (y/n)")
				if yesOrNo() {
					draft.Commit()
					fmt.Println("returning to main page")
					return
				}
				fmt.Println("changes aborted, returning to main page")
				return
			}
		}
		fields= strings.Fields(input)
		if len(fields)== 0{
			return
		}
		switch strings.ToLower(fields[0]){
			case "abort", "cancel":
			fmt.Println("draft cancelled")
			return
			case "del", "delete", "remove", "rm":
			draft.Delete(fields [1:])
			case "add", "search":
			draft.Search(fields[1:])
			case "commit", "save", "done", "okay":
			draft.Commit()
			if len(fields) > 1{
				option:= concat(fields[1:])
				switch strings.ToLower(option){
					case "and return", "and exit", "return", "exit":
					fmt.Println("done editing, returning")
					default:
				}
			}
			case "help", "h":
			helpDraft()
			case "review", "show":
			draft.Review()
			case "rename":
			draft.Rename(fields[1:])
			case "next": 
			changePage(1)
			case "prev", "previous":
			changePage(-1)
			case "list":
			draft.Display(currentPage)
			default:
			fmt.Printf("unknown command %q, type 'help' for help\n", fields[0])
		}
		
	}
}

func(p *Playlist) Search(args []string) {
	if len(args)==0{
		fmt.Println("missing argument 'song' for àdd`")
		return
	}
	temp:= concat(args)
	var artist, name string
	if strings.Contains(temp, "::"){
		split:= strings.Split(temp, "::")
		if len(split) == 2{
			name= strings.TrimSpace(split[0])
			artist= strings.TrimSpace(split[1])
		}else{
			name= strings.TrimSpace(temp)
		}
	}else if strings.Contains(temp, "-"){
		split:= strings.Split(temp, "-")
		if len(split)==2{
			name= strings.TrimSpace(split[0])
			artist= strings.TrimSpace(split[1])
		}else{
			name= strings.TrimSpace(temp)
		}
	}else{
		name= temp
	}
	query:= "track:" + name
	if artist!= ""{
		query+= " artist:" + artist
	}
	results, err:= client.Search(query, spotify.SearchTypeTrack)
	if err!=nil{
		fmt.Printf("no results found for %q\n", query)
		return
	}
	if results.Tracks== nil{
		fmt.Printf("no results found for %q\n", query)
		return
	}
	tracks:= results.Tracks.Tracks
	if len(tracks)==0{
		fmt.Printf("no results found for %q\n", query)
		return
	}
	length:= len(tracks)
	if length> 15{
		length= 15
		tracks= tracks[:length]
	}
	for i, t:= range tracks{
		var artists string
		for _, ar:= range t.Artists{
			artists+= ar.Name + ", "
		}
		fmt.Printf("%d- %s by %s\n",
		i,
		t.Name,
		artists)
	}
	fmt.Printf("choose track to add to %s, (0-%d), -1 or blank to cancel\n", selectedSimple.Name, length-1)
	input:= ""
	for{
		input= prompt()
		if input== "-1" || input== ""{
			fmt.Println("cancelled")
			return
		}
		index, err:= strconv.Atoi(input)
		if err!=nil{
			fmt.Println("invalid input, enter again:")
			continue
		}
		if index <0 || index>= length{
			fmt.Printf("invalid input, enter 0-%d, -1 or blank to cancel:\n", length-1)
			continue
		}
		p.Add(tracks[index])
		return
	}
}

func (p *Playlist) Add(track spotify.FullTrack) {
	id:= track.ID.String()
	var artists []string
	for _, ar:= range track.Artists{
		artists= append(artists, ar.Name)
	}
	for _, t:= range p.Tracks{
		if t.ID== id{
			fmt.Printf("%s already has the track %s\n", p.Name, t.Name)
			return
		}
	}
	for _, t:= range p.addCache{
		if t.ID== id{
			fmt.Printf("the playlist %s already has the song %s\n", p.Name, t.Name)
			return
		}
	}
	for i, t:= range p.removeCache{
		if t.ID==id{
			fmt.Printf("added %s to %s\n", t.Name, p.Name)
			p.Tracks= append(p.Tracks, t)
			p.removeCache= append(p.removeCache[:i], p.removeCache[i+1:]...)
			return
		}
	}
	p.addCache= append(p.addCache, Track{
		Name: track.Name,
		ID: id,
		Artists: artists,
	})
	fmt.Printf("added %s to %s\n", track.Name, p.Name)
}

func(p *Playlist) Commit() {
	if len(p.addCache)== 0 && len(p.removeCache)==0{
		return
	}
	var dels []spotify.ID
	var adds []spotify.ID
	for _, t:= range p.addCache{
		adds= append(adds, spotify.ID(t.ID))
	}
	for _, t:= range p.removeCache{
		dels= append(dels, spotify.ID(t.ID))
	}
	if len(adds)>0{
		_, err:= client.AddTracksToPlaylist(spotify.ID(p.ID), adds...)
		if err!=nil{
			fmt.Fprintf(os.Stderr, "error adding tracks to %s: %s\n", p.Name, err)
			os.Exit(1)
		}
		fmt.Printf("successfully committed new tracks to %s\n", p.Name)
	}
	if len(dels)>0{
		_, err:= client.RemoveTracksFromPlaylist(spotify.ID(p.ID), dels...)
		if err!=nil{
			fmt.Fprintf(os.Stderr, "error removing tracks from %s: %s\n", p.Name, err)
			os.Exit(1)
		}
		fmt.Printf("successfully committed removing tracks from %s\n", p.Name)
	}
	fmt.Println("done")
}

func(p *Playlist) Review() {
	if len(p.addCache)==0 && len(p.removeCache)==0{
		fmt.Printf("there you do not have any uncommitted changes to %s\n", p.Name)
		return
	}
	if len(p.addCache) >0{
		fmt.Printf("%d pending additions:\n", len(p.addCache))
		for _, t:= range p.addCache{
			fmt.Printf("\t#%s\n", t)
		}
	}
	if len(p.removeCache)> 0{
		fmt.Printf("%d pending remove operations:\n", len(p.removeCache))
		for _, t:= range p.removeCache{
			fmt.Printf("\t#%s\n", t)
		}
	}
}

func(p *Playlist) Display(page int) {
	if page< 0{
		return
	}
	start:= page*40
	end:= start+ 40
	if len(p.Tracks) < end{
		end= len(p.Tracks)
	}
	for i:= start; i< end; i++{
		fmt.Printf("%d- %s\n", i, p.Tracks[i])
	}
}

func(p *Playlist) Delete(args []string) {
	if len(args)==0{
		fmt.Println("missing argument 'index' for `remove`")
		return
	}
	var numbers []int
	for _, s:= range args{
		num, err:= strconv.Atoi(s)
		if err!=nil{
			fmt.Printf("not a number: %s\n", s)
			continue
		}
		if num < 0 || num >= len(p.Tracks) {
			fmt.Printf("index out of range: %s (%d max)\n", s, len(p.Tracks)-1)
			continue
		}
		numbers= append(numbers, num)
	}
	if len(numbers)==0{
		fmt.Println("no changes made.")
		return
	}
	var temp []Track
	LOOP:
	for i, t:= range p.Tracks{
		for _, n:= range numbers{
			if n== i{
				p.removeCache= append(p.removeCache, t)
				continue LOOP
			}
		}
		temp= append(temp, t)
	}
	p.Tracks= temp
	fmt.Printf("%d tracks queued for removal\n", len(numbers))
}

func(p *Playlist) Rename(args []string) {
	if len(args)==0{
		fmt.Println("playlist name can't be empty")
		return
	}
	name:= concat(args)
	fmt.Printf("are you sure you want to rename %s to %s? (y/n):\n", p.Name, name)
	if yesOrNo() {
		err:= client.ChangePlaylistName(spotify.ID(p.ID), name)
		if err!=nil{
			fmt.Printf("error renaming %s: %s\n", p.Name, err)
			return
		}
		p.Name= name
		fmt.Println("rename successful")
		return
	}
	fmt.Println("cancelled")
}

func helpDraft() {
	msg:= `commands:
	#add/search [track]
	searches for a track, if desired, adds to the playlist.
	name::artist or name - artist are accepted as well
	
	#delete/remove [number]
	removes one or more tracks from the playlist
	values must be separated with a whitespace
	only valid values will be removed
	
	#commit/save [and return]
	if entered without arguments, commits the changes ot the live playlist
	if specified with "and return", the playlist will be committed and you'll return to the main page
	
	#rename <new name>
	renames the playlist
	
	#abort/cancel
	aborts all the changes and returns to the main menu
	
	#review
	review the uncommitted changes
	
	#list
	print the current page
	
	#next/n
	show the next page (each page contains up to 40 tracks)
	
	#previous/prev
	show the previous page
	`
	fmt.Println(msg)
}

func parseLiveCommand(s string) {
	fields:= strings.Fields(s)
	if len(fields) == 0{
		fmt.Print("\n")
		return
	}
	switch strings.ToLower(fields[0]){
		case "select", "choose", "sel":
		choosePlaylist(fields[1:])
		case "edit", "e":
		editSelectedPlaylist()
		case "new", "create":
		createPlaylist()
		case "delete", "del", "rm", "remove":
		deletePlaylist()
		case "load", "cache":
		loadCache(fields[1:])
		case "sync":
		syncCache()
		case "h", "help":
		helpLive(fields[1:])
		case "show", "display":
		show(fields[1:])
		case "caches":
		listCaches()
		default:
		fmt.Printf("unknown command %q\n", concat(fields[1:]))
	}
	
}

func createPlaylist() {
	fmt.Println("creating new playlist")
	fmt.Println("playlist name:")
	name:= strings.TrimSpace(prompt())
	fmt.Println("playlist description:")
	description:= strings.TrimSpace(prompt())
	fmt.Println("should the playlist be public? (y/n):")
	publ:= true
	if !yesOrNo(){
		publ= false
	}
	fmt.Printf("#name: %s\n#description: %s\n#public: %t\nare you sure you want to create the playlist? (y/n)\n",
	name, description, publ)
	if !yesOrNo(){
		fmt.Println("cancelled")
		return
	}
	_, err:= client.CreatePlaylistForUser(user.ID, name, description, publ)
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error creating new playlist: %s\n", err)
		return
	}
	fmt.Printf("created playlist %s, select it with `select`\n", name)
}

func loadCache(args []string) {
	if len(caches)==0{
		fmt.Println("you don't have any caches, use `libman search` to manage local caches")
		return
	}
	if len(args)>0{
		name:= concat(args)
		for _, cache:= range caches{
			if strings.EqualFold(name, cache.Name){
				selectedCache= cache
				fmt.Printf("loaded cache %s\n", cache.Name)
				return
			}
		}
		fmt.Printf("there are no caches with the name %s\n", name)
		return
	}
	
	for i, cache:= range caches{
		fmt.Printf("%d- %s, %d tracks\n", i, cache.Name, len(cache.Tracks))
	}
	fmt.Printf("choose cache (0-%d), -1 or blank to cancel\n", len(caches)-1)
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
		if index<0 || index>= len(caches){
			fmt.Printf("invalid input, enter 0-%d, blank or -1 to cancel:\n", len(caches)-1)
			continue
		}
		selectedCache= caches[index]
		fmt.Printf("loaded cache %s\n", selectedCache.Name)
		return
 	}
}

func liveCleanup(){
	defer db.Close()
	if db== nil{
		fmt.Println("db is nil, what the hell")
		os.Exit(1)
	}
	if client== nil{
		db.Close()
		fmt.Println("warning, the client is nil")
		os.Exit(1)
	}
	// save caches
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
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error updating cache database: %s\n", err)
	}
	// save token for later use
	
	token, err:= client.Token()
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error saving token: %s\n", err)
		db.Close()
		os.Exit(1)
	}
	data, err:= json.MarshalIndent(token, "", "\t")
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error marshaling token: %s\n", err)
		db.Close()
		os.Exit(1)
	}
	err= db.Update(func(tx *bolt.Tx)error{
		b:= tx.Bucket([]byte("token"))
		err:= b.Put([]byte("token"), data)
		return err
	})
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error saving token to db: %s\n", err)
		db.Close()
		os.Exit(1)
	}
	
}

func helpLive(_ []string) {
	msg:= `commands:
	#new
	create a new playlist
	
	#choose/select
	choose a playlist
	
	#load/cache
	load a cache
	
	#list cache|playlist
	list the selected playlists or caches tracks
	
	#caches
	show the list of caches
	`
	fmt.Println(msg)
}

func show(args []string) {
	if len(args) != 0{
		switch strings.ToLower(args[0]){
			case "cache", "c", "cac":
			showCache(args[1:])
			case "p", "playlist", "play", "pl":
			showPlaylist(args[1:])
			default:
			fmt.Printf("unknown argument %s, arguments are:\nplaylist (p, pl, play)\ncache (c, cac)\n", args[0])
			return
		}
	}
}