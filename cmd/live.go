package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/insomnimus/libman/userdir"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/zmb3/spotify"
)

const redirectURI = "http://localhost:8080/callback"

var (
	user *spotify.PrivateUser
	// all the permissions
	auth = spotify.NewAuthenticator(redirectURI,
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

var (
	simplePlaylists []spotify.SimplePlaylist
	selectedSimple  *spotify.SimplePlaylist
)

func init() {
	// check if config contains client id and secret
	configPath := userdir.GetConfigHome() + "/libman.config"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// create an empty config file
		configDir := userdir.GetConfigHome()
		err := os.MkdirAll(configDir, 0600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error initializing the config path: %s\n", err)
			os.Exit(2)
		}
		err = ioutil.WriteFile(configPath, []byte(TemplateConfig), 0600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating the template config file: %s\n", err)
			os.Exit(2)
		}
		id := os.Getenv("SPOTIFY_ID")
		secret := os.Getenv("SPOTIFY_SECRET")
		if id == "" || secret == "" {
			fmt.Fprintf(os.Stderr, "please either set the 'SPOTIFY_ID' and 'SPOTIFY_SECRET' env variables or edit the config file located at %s\n", configPath)
			os.Exit(2)
		}
	}
	// read config
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading the config file: %s\n", err)
		os.Exit(2)
	}
	lines := strings.Split(string(data), "\n")
	var id, secret string
	for _, l := range lines {
		if len(l) <= 10 {
			continue
		}
		if l[0] == '#' || l[0] == ' ' {
			continue
		}
		if l[:2] == "id" {
			temp := strings.SplitN(l, " ", 2)
			if len(temp) != 2 {
				continue
			}
			id = temp[1]
		}
		if l[:6] == "secret" {
			temp := strings.SplitN(l, " ", 2)
			if len(temp) != 2 {
				continue
			}
			secret = temp[1]
		}
	}
	if id == "" {
		id := os.Getenv("SPOTIFY_ID")
		if id == "" {
			fmt.Fprintf(os.Stderr, "please set your SPOTIFY_ID variable either in %s (gihest priority) or in your environment\n", configPath)
			os.Exit(2)
		}
	}
	if secret == "" {
		secret := os.Getenv("SPOTIFY_SECRET")
		if secret == "" {
			fmt.Fprintf(os.Stderr, "please set your SPOTIFY_SECRET variable either in %s (gihest priority) or in your environment\n", configPath)
			os.Exit(2)
		}
	}
	auth.SetAuthInfo(id, secret)
}

func checkToken() {
	if db == nil {
		dbptr, err := bolt.Open(dbName, 0600, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening db: %s\n", err)
			os.Exit(1)
		}
		db = dbptr
	}
	var data []byte
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("token"))
		data = b.Get([]byte("token"))
		return nil
	})
	if data == nil {
		authorize()
		return
	}
	var token oauth2.Token
	err := json.Unmarshal(data, &token)
	if err != nil {
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
	clt := <-ch

	// use the client to make calls that require authorization
	usr, err := clt.CurrentUser()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error authorizing: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("logged in!")
	client = clt
	user = usr
	token, err := client.Token()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error saving access token: %s\n", err)
		os.Exit(1)
	}
	data, err := json.Marshal(token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling token: %s\n", err)
		os.Exit(1)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("token"))
		err := b.Put([]byte("token"), data)
		return err
	})
	if err != nil {
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
	clt := auth.NewClient(token)
	client = &clt
	usr, err := client.CurrentUser()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	user = usr
	fmt.Printf("welcome %s\n", user.DisplayName)
}

func choosePlaylist(args []string) {
	page, err := client.CurrentUsersPlaylists()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching users playlists: %s\n", err)
		os.Exit(1)
	}
	if len(page.Playlists) == 0 {
		fmt.Println("you don't seem to have any playlist, create one with `new`")
		return
	}
	simplePlaylists = page.Playlists
	if len(args) != 0 {
		name := concat(args)
		for _, pl := range simplePlaylists {
			if strings.EqualFold(pl.Name, name) {
				fmt.Printf("selected playlist %s\n", name)
				selectedSimple = &pl
				return
			}
		}
		fmt.Printf("there are no playlists with the name %s, use `new` to create one\n", name)
		return
	}
	for i, pl := range simplePlaylists {
		fmt.Printf("%d- %s\n", i, pl.Name)
	}
	fmt.Printf("choose playlist (0-%d), -1 or blank to cancel\n", len(simplePlaylists)-1)
	var input string
	for {
		input = prompt()
		if input == "" || input == "-1" {
			fmt.Println("cancelled")
			return
		}
		index, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("invalid input, enter again:")
			continue
		}
		if index < 0 || index >= len(simplePlaylists) {
			fmt.Printf("invalid input, enter 0-%d, -1 or blank to cancel\n", len(simplePlaylists)-1)
			continue
		}
		selectedSimple = &simplePlaylists[index]
		fmt.Printf("selected %s\n", selectedSimple.Name)
		return
	}
}

func editSelectedPlaylist() {
	if selectedSimple == nil {
		fmt.Println("you need to choose a playlist with `select`")
		return
	}
	fmt.Println("enter blank or -1 to go back")
	page, err := client.GetPlaylistTracks(selectedSimple.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching the list of tracks in %s:\n%s\n", selectedSimple.Name, err)
		os.Exit(1)
	}
	draft := &Playlist{Name: selectedSimple.Name, ID: string(selectedSimple.ID)}
	currentPage := 0
	if len(page.Tracks) > 0 {
		for _, tr := range page.Tracks {
			track := Track{
				Name: tr.Track.Name,
				ID:   string(tr.Track.ID),
			}
			for _, art := range tr.Track.Artists {
				track.Artists = append(track.Artists, art.Name)
			}
			draft.Tracks = append(draft.Tracks, track)
		}
	}
	draft.Display(currentPage)
	changePage := func(step int) {
		if currentPage+step < 0 {
			fmt.Println("you can't go back page 1")
			return
		}
		if (currentPage+step-1)*40 >= len(draft.Tracks) {
			fmt.Println("this is already the last page")
			return
		}
		currentPage += step
		draft.Display(currentPage)
	}
	var input string
	var fields []string
LOOP:
	for {
		input = prompt()
		if input == "-1" || input == "" {
			if len(draft.addCache) != 0 || len(draft.removeCache) != 0 {
				fmt.Println("you have unsaved changes, commit them? (yes/no/cancel))")
				yea := prompt()
				switch strings.ToLower(yea) {
				case "y", "yea", "yes", "save":
					draft.Commit()
					fmt.Println("returning")
					return
				case "n", "no", "nope", "nah":
					fmt.Println("changes aborted, returning")
					return
				default:
					fmt.Println("cancelled")
					continue LOOP
				}
			}
		}
		fields = strings.Fields(input)
		if len(fields) == 0 {
			return
		}
		switch strings.ToLower(fields[0]) {
		case "abort", "cancel":
			fmt.Println("draft cancelled")
			return
		case "del", "delete", "remove", "rm":
			draft.Delete(fields[1:])
		case "add", "search":
			draft.Search(fields[1:])
		case "commit", "save", "done", "okay":
			draft.Commit()
			if len(fields) > 1 {
				option := concat(fields[1:])
				switch strings.ToLower(option) {
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

func (p *Playlist) Search(args []string) {
	if len(args) == 0 {
		fmt.Println("missing argument 'song' for àdd`")
		return
	}
	temp := concat(args)
	var artist, name string
	if strings.Contains(temp, "::") {
		split := strings.Split(temp, "::")
		if len(split) == 2 {
			name = strings.TrimSpace(split[0])
			artist = strings.TrimSpace(split[1])
		} else {
			name = strings.TrimSpace(temp)
		}
	} else if strings.Contains(temp, "-") {
		split := strings.Split(temp, "-")
		if len(split) == 2 {
			name = strings.TrimSpace(split[0])
			artist = strings.TrimSpace(split[1])
		} else {
			name = strings.TrimSpace(temp)
		}
	} else {
		name = temp
	}
	query := "track:" + name
	if artist != "" {
		query += " artist:" + artist
	}
	results, err := client.Search(query, spotify.SearchTypeTrack)
	if err != nil {
		fmt.Printf("no results found for %q\n", query)
		return
	}
	if results.Tracks == nil {
		fmt.Printf("no results found for %q\n", query)
		return
	}
	tracks := results.Tracks.Tracks
	if len(tracks) == 0 {
		fmt.Printf("no results found for %q\n", query)
		return
	}
	length := len(tracks)
	if length > 15 {
		length = 15
		tracks = tracks[:length]
	}
	for i, t := range tracks {
		var artists string
		for _, ar := range t.Artists {
			artists += ar.Name + ", "
		}
		fmt.Printf("%d- %s by %s\n",
			i,
			t.Name,
			artists)
	}
	fmt.Printf("choose track to add to %s, (0-%d), -1 or blank to cancel\n", selectedSimple.Name, length-1)
	input := ""
	for {
		input = prompt()
		if input == "-1" || input == "" {
			fmt.Println("cancelled")
			return
		}
		index, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("invalid input, enter again:")
			continue
		}
		if index < 0 || index >= length {
			fmt.Printf("invalid input, enter 0-%d, -1 or blank to cancel:\n", length-1)
			continue
		}
		p.Add(tracks[index])
		return
	}
}

func (p *Playlist) Add(track spotify.FullTrack) {
	id := track.ID.String()
	var artists []string
	for _, ar := range track.Artists {
		artists = append(artists, ar.Name)
	}
	for _, t := range p.Tracks {
		if t.ID == id {
			fmt.Printf("%s already has the track %s\n", p.Name, t.Name)
			return
		}
	}
	for _, t := range p.addCache {
		if t.ID == id {
			fmt.Printf("the playlist %s already has the song %s\n", p.Name, t.Name)
			return
		}
	}
	for i, t := range p.removeCache {
		if t.ID == id {
			fmt.Printf("added %s to %s\n", t.Name, p.Name)
			p.Tracks = append(p.Tracks, t)
			p.removeCache = append(p.removeCache[:i], p.removeCache[i+1:]...)
			return
		}
	}
	p.addCache = append(p.addCache, Track{
		Name:    track.Name,
		ID:      id,
		Artists: artists,
	})
	fmt.Printf("added %s to %s\n", track.Name, p.Name)
}

func (p *Playlist) Commit() {
	if len(p.addCache) == 0 && len(p.removeCache) == 0 {
		return
	}
	var dels []spotify.ID
	var adds []spotify.ID
	for _, t := range p.addCache {
		adds = append(adds, spotify.ID(t.ID))
	}
	for _, t := range p.removeCache {
		dels = append(dels, spotify.ID(t.ID))
	}
	if len(adds) > 0 {
		_, err := client.AddTracksToPlaylist(spotify.ID(p.ID), adds...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error adding tracks to %s: %s\n", p.Name, err)
			return
		}
		fmt.Printf("successfully committed new tracks to %s\n", p.Name)
		p.addCache = nil
	}
	if len(dels) > 0 {
		_, err := client.RemoveTracksFromPlaylist(spotify.ID(p.ID), dels...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error removing tracks from %s: %s\n", p.Name, err)
			return
		}
		fmt.Printf("successfully committed removing tracks from %s\n", p.Name)
		p.removeCache = nil
	}
	fmt.Println("done")
}

func (p *Playlist) Review() {
	if len(p.addCache) == 0 && len(p.removeCache) == 0 {
		fmt.Printf("there you do not have any uncommitted changes to %s\n", p.Name)
		return
	}
	if len(p.addCache) > 0 {
		fmt.Printf("%d pending additions:\n", len(p.addCache))
		for _, t := range p.addCache {
			fmt.Printf("\t#%s\n", t)
		}
	}
	if len(p.removeCache) > 0 {
		fmt.Printf("%d pending remove operations:\n", len(p.removeCache))
		for _, t := range p.removeCache {
			fmt.Printf("\t#%s\n", t)
		}
	}
}

func (p *Playlist) Display(page int) {
	if page < 0 {
		return
	}
	start := page * 40
	end := start + 40
	if len(p.Tracks) < end {
		end = len(p.Tracks)
	}
	for i := start; i < end; i++ {
		fmt.Printf("%d- %s\n", i, p.Tracks[i])
	}
}

func (p *Playlist) Delete(args []string) {
	if len(args) == 0 {
		fmt.Println("missing argument 'index' for `remove`")
		return
	}
	var numbers []int
	for _, s := range args {
		num, err := strconv.Atoi(s)
		if err != nil {
			fmt.Printf("not a number: %s\n", s)
			continue
		}
		if num < 0 || num >= len(p.Tracks) {
			fmt.Printf("index out of range: %s (%d max)\n", s, len(p.Tracks)-1)
			continue
		}
		numbers = append(numbers, num)
	}
	if len(numbers) == 0 {
		fmt.Println("no changes made.")
		return
	}
	var temp []Track
LOOP:
	for i, t := range p.Tracks {
		for _, n := range numbers {
			if n == i {
				p.removeCache = append(p.removeCache, t)
				continue LOOP
			}
		}
		temp = append(temp, t)
	}
	p.Tracks = temp
	fmt.Printf("%d tracks queued for removal\n", len(numbers))
}

func (p *Playlist) Rename(args []string) {
	if len(args) == 0 {
		fmt.Println("playlist name can't be empty")
		return
	}
	name := concat(args)
	fmt.Printf("are you sure you want to rename %s to %s? (y/n):\n", p.Name, name)
	if yesOrNo() {
		err := client.ChangePlaylistName(spotify.ID(p.ID), name)
		if err != nil {
			fmt.Printf("error renaming %s: %s\n", p.Name, err)
			return
		}
		p.Name = name
		fmt.Println("rename successful")
		return
	}
	fmt.Println("cancelled")
}

func helpDraft() {
	msg := `commands:
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

func createPlaylist() {
	fmt.Println("creating new playlist")
	fmt.Println("playlist name:")
	name := strings.TrimSpace(prompt())
	fmt.Println("playlist description:")
	description := strings.TrimSpace(prompt())
	fmt.Println("should the playlist be public? (y/n):")
	publ := true
	if !yesOrNo() {
		publ = false
	}
	fmt.Printf("#name: %s\n#description: %s\n#public: %t\nare you sure you want to create the playlist? (y/n)\n",
		name, description, publ)
	if !yesOrNo() {
		fmt.Println("cancelled")
		return
	}
	_, err := client.CreatePlaylistForUser(user.ID, name, description, publ)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating new playlist: %s\n", err)
		return
	}
	fmt.Printf("created playlist %s\n", name)
}

func renamePlaylist(args []string) {
	if len(args) == 0 {
		fmt.Println("usage: 'rename old >> new'")
		return
	}
	text := concat(args)
	if !strings.Contains(text, ">>") {
		fmt.Println("usage: 'rename old >> new'")
		return
	}
	temp := strings.Split(text, ">>")
	if len(temp) != 2 {
		fmt.Println("usage: 'rename old >> new'\nthe '>>' can only be used once in one command")
		return
	}
	oldName := strings.TrimSpace(temp[0])
	newName := strings.TrimSpace(temp[1])
	pls, err := getPlaylists()
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, pl := range pls {
		if strings.EqualFold(pl.Name, oldName) {
			err := client.ChangePlaylistName(spotify.ID(pl.ID), newName)
			if err != nil {
				fmt.Printf("couldn't rename %s: %s\n", pl.Name, err)
				return
			}
			fmt.Printf("successfully renamed %s to %s\n", oldName, newName)
			return
		}
	}
	fmt.Printf("you don't seem to have any playlist by the name %s\n", oldName)
}
