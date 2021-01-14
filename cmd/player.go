package cmd

import (
"github.com/zmb3/spotify"
"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
"os/signal"
"os"
"strconv"
"strings"
//"github.com/zmb3/spotify"
	"github.com/spf13/cobra"
)

var(
playerVolume= 80
isPlaying bool
shuffleState bool
)

var playerCmd = &cobra.Command{
	Use:   "player",
	Short: "start the player",
	Long: `starts the player mode`,
	Run: func(cmd *cobra.Command, args []string) {
		startPlayerSession()
	},
}

func init() {
	rootCmd.AddCommand(playerCmd)	
}

func startPlayerSession() {
	COMMAND= "player"
	signal.Notify(terminator, os.Interrupt)
	checkToken()
	initPlayer()
	//fmt.Printf("welcome %s\n", user.DisplayName)
	for{
		parsePlayerCommand(prompt())
	}
}

func initPlayer(){
	plState, err:= client.PlayerState()
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error fetching player state: %s\n", err)
		os.Exit(2)
	}
	shuffleState= plState.ShuffleState
	isPlaying= plState.Playing
	curPlaying:= plState.Item
	if curPlaying!=nil{
		artists:= ""
		for _, art:= range curPlaying.Artists{
			artists+= art.Name + ", "
		}
		fmt.Printf("currently playing:\n%s by %s\n(pause=%t, volume= %d%%, shuffle=%t)\n",
		curPlaying.Name,
		artists,
		!isPlaying,
		playerVolume,
		shuffleState)
		return
	}
	fmt.Printf("shuffle=%t\n", shuffleState)
}

func saveCurrentlyPlaying(args []string) {
	playingTrack, err:= currentlyPlayingTrack()
	if err!=nil{
		fmt.Printf("failed to fetch currently playing info: %s\n", err)
		return
	}
	if len(args)==0{
		playlists, err:= getPlaylists()
		if err!=nil{
			fmt.Fprintln(os.Stderr, err)
			return
		}
		for i, pl:= range playlists{
			fmt.Printf("%d- %s\n", i, pl.Name)
		}
		fmt.Printf("add to which playlist? (0-%d), -1 or blank to cancel\n", len(playlists)-1)
		var input string
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
			if index<0 || index>= len(playlists){
				fmt.Printf("invalid input, enter 0-%d, blank or -1 to cancel:\n", len(playlists)-1)
				continue
			}
			playlists[index].addCache= append(playlists[index].addCache, playingTrack)
			playlists[index].Commit()
			fmt.Printf("done")
			return
		}
	}
	name:= concat(args)
	pls, err:= getPlaylists()
	if err!=nil{
		fmt.Fprintln(os.Stderr, err)
		return
	}
	for _, p:= range pls{
		if strings.EqualFold(p.Name, name){
			p.addCache= append(p.addCache, playingTrack)
			p.Commit()
			fmt.Println("done")
			return
		}
	}
	fmt.Printf("no playlist found by the name %s\n", name)
}

func playPrev() {
	err:= client.Previous()
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}
	fmt.Println("playing previous track")
}

func playNext() {
	err:= client.Next()
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}
	fmt.Println("playing the next track")
}

func currentlyPlayingTrack() (Track, error) {
	var track Track
	cp, err:= client.PlayerCurrentlyPlaying()
	if err!=nil{
		return track, err
	}
	if !cp.Playing{
		return track, fmt.Errorf("not playing anything")
	}
	track.ID= string(cp.Item.ID)
	for _, art:= range cp.Item.Artists{
		track.Artists= append(track.Artists, art.Name)
	}
	track.Name= cp.Item.Name
	return track, nil
}

func changeVolume(arg string) {
	amount, err:= strconv.Atoi(arg)
	if err!=nil{
		fmt.Printf("invalid input %s\n", arg)
		return
	}
	if playerVolume ==0 && amount <=0{
		fmt.Println("muted")
		return
	}
	if playerVolume== 100 && amount >=0{
		fmt.Println("max")
		return
	}
	playerVolume+= amount
	if playerVolume <0 {
		playerVolume= 0
	}
	if playerVolume >100{
		playerVolume= 100
	}
	err= client.Volume(playerVolume)
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error setting the volume: %s\n", err)
		return
	}
}

func toggle() {
	if isPlaying{
		err:= client.Pause()
		if err!=nil{
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return
		}
		isPlaying= false
		return
	}
	err:= client.Play()
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}
	isPlaying= true
}

func playTrack(arg string) {
	results, err:= searchAll(arg)
	if err!=nil{
		fmt.Printf("error: %s\n", err)
		return
	}
	results.chooseInteractive()
}

func volume(args []string) {
	if len(args)==0{
		fmt.Printf("the volume is %d%%\n", playerVolume)
		return
	}
	arg:= concat(args)
	arg= strings.Replace(arg, " ", "", -1)
	if strings.Contains(arg, "+") || strings.Contains(arg, "-"){
		changeVolume(arg)
		return
	}
	amount, err:= strconv.Atoi(arg)
	if err!=nil{
		fmt.Printf("invalid argument for volume %q\n", arg)
		return
	}
	if amount <=0{
		changeVolume("-100")
		return
	}
	if amount >= 100{
		changeVolume("+100")
		return
	}
	if amount== playerVolume{
		return
	}
	if amount > playerVolume{
	changeVolume(fmt.Sprintf("+%d", amount-playerVolume))
	return
	}
	
	changeVolume(fmt.Sprintf("-%d", playerVolume-amount))
}

func playerCleanup() {
	if db== nil{
		fmt.Println("fatal error: db is nil")
		os.Exit(1)
		}
		// when hit ctrl-c it switches play state, so do it again
		toggle()
		// save the token
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
	db.Close()
	os.Exit(0)
}

func showCurrentlyPlaying() {
	tr, err:= currentlyPlayingTrack()
	if err!=nil{
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Printf("currently playing %s\nshuffle=%t\n", tr, shuffleState)
}

type SearchResult struct{
	Name, Artists string
	URI *spotify.URI
	URIs []spotify.URI
	Type string
}

type SearchResults []SearchResult

func(sr SearchResult) Play() {
	var opt spotify.PlayOptions
	if sr.URI!= nil{
		// means non-track
		opt.PlaybackContext= sr.URI
	}else{
		// means track(s)
		opt.URIs= sr.URIs
	}
	err:= client.PlayOpt(&opt)
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}
	fmt.Printf("playing %s\n", sr.Name)
}

func(sr SearchResult) String() string{
	switch strings.ToLower(sr.Type) {
		case "track", "song":
		return fmt.Sprintf("track - %s by %s", sr.Name, sr.Artists)
		case "playlist":
		return fmt.Sprintf("playlist - %s", sr.Name)
		case "artist":
		return fmt.Sprintf("artist - %s", sr.Name)
		case "album":
		return fmt.Sprintf("album - %s by %s", sr.Name, sr.Artists)
		default:
		return fmt.Sprintf("result - %s", sr.Name)
	}
	return fmt.Sprintf("result - %s", sr.Name)
}

func(srs *SearchResults) Add(res ...SearchResult) {
	*srs= append(*srs, res...)
}

func(srs *SearchResults) chooseInteractive() {
	if len(*srs)==0{
		return
	}
	// to hold max 5 of each result type
	var displays SearchResults
	var art, alb, pla, tra SearchResults
	for _, res:= range *srs{
		switch strings.ToLower(res.Type){
			case "playlist":
			pla.Add(res)
			case "track", "song":
			tra.Add(res)
			case "album":
			alb.Add(res)
			case "artist":
			art.Add(res)
			default:
		}
	}
	if len(tra) >=5{
		displays.Add(tra[:5]...)
	}else{
		displays.Add(tra...)
	}
	if len(pla)>= 5{
		displays.Add(pla[:5]...)
	}else{
		displays.Add(pla...)
	}
	if len(art) >= 5{
		displays.Add(art[:5]...)
	}else{
		displays.Add(art...)
	}
	if len(alb) >= 5{
		displays.Add(alb[:5]...)
	}else{
		displays.Add(alb...)
	}
	for i, res:= range displays{
		fmt.Printf("%d- %s\n", i, res)
	}
	var input string
	fmt.Printf("choose (0-%d), blank or -1 to cancel:\n", len(displays))
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
		if index <0 || index >= len(displays){
			fmt.Printf("invalid input, enter between 0-%d, blank or -1 to cancel\n", len(displays)-1)
			continue
		}
		displays[index].Play()
		return
	}
}

func searchAll(arg string)  (SearchResults, error){
	defer IdentifyPanic()
	if arg== ""{
		return nil, fmt.Errorf("missing argument `query` for search")
	}
	var query string
	if strings.Contains(arg, "::"){
		split:= strings.Split(arg, "::")
		if len(split) == 2{
			query= fmt.Sprintf("track:%s artist:%s",
			strings.TrimSpace(split[0]),
			strings.TrimSpace(split[1]))
		}else{
			query= arg
		}
	}else if strings.Contains(arg, "-"){
		split:= strings.Split(arg, "-")
		if len(split)==2{
			query= fmt.Sprintf("track:%s artist:%s",
			strings.TrimSpace(split[0]),
			strings.TrimSpace(split[1]))
		}else{
			query= arg
		}
	}else{
		query= arg
	}
	// TODO: change the client here
	page, err:= client.Search(query, spotify.SearchTypePlaylist|spotify.SearchTypeTrack|spotify.SearchTypeArtist|spotify.SearchTypeAlbum)
	if err!=nil{
		return nil, err
	}
	var results SearchResults
	if page.Tracks!= nil && len(page.Tracks.Tracks)>0{
		for _, t:= range page.Tracks.Tracks{
			artists:= ""
			for _, art:= range t.Artists{
				artists+= art.Name + ", "
			}
			results.Add(SearchResult{
				Name: t.Name,
				Artists: artists,
				Type: "track",
				URIs: []spotify.URI{t.URI},
			})
		}
	}
	if page.Playlists != nil && len(page.Playlists.Playlists)>0{
		for _, pl:= range page.Playlists.Playlists{
			results.Add(SearchResult{
				Name: pl.Name,
				URI: &pl.URI,
				Type: "playlist",
			})
		}
	}
	if page.Artists!= nil && len(page.Artists.Artists) > 0{
		for _, art:= range page.Artists.Artists{
			results.Add(SearchResult{
				Name: art.Name,
				URI: &art.URI,
				Type: "artist",
			})
		}
	}
	if page.Albums.Albums!=nil && len(page.Albums.Albums)>0{
		for _, alb:= range page.Albums.Albums{
			artists:= ""
			for _, art:= range alb.Artists{
				artists+= art.Name + ", "
			}
			results.Add(SearchResult{
				Name: alb.Name,
				Artists: artists,
				URI: &alb.URI,
				Type: "album",
			})
		}
	}
	if len(results)== 0{
		return nil, fmt.Errorf("no result found for %s", query)
	}
	return results, nil
}

func playUserPlaylist(args []string) {
	page, err:= client.CurrentUsersPlaylists()
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error fetching users playlists: %s\n", err)
		return
	}
	if len(page.Playlists)==0{
		fmt.Println("you don't seem to have any playlists")
		return
	}
	pls:= page.Playlists
	
	if len(args)==0{
		for i, p:= range pls{
		fmt.Printf("%d- %s\n", i, p.Name)
	}
		fmt.Printf("play which one? (0-%d), blank or -1 to cancel:\n", len(pls)-1)
	input:= ""
	for{
		input= prompt()
		if input== "-1" || input== ""{
			fmt.Println("cancelled")
			return
		}
		index, err:= strconv.Atoi(input)
		if err != nil{
			fmt.Println("invalid input, enter again:")
			continue
		}
		if index<0 || index>= len(pls){
			fmt.Printf("invalid input, enter 0-%d, blank or -1 to cancel:\n", len(pls)-1)
			continue
		}
		temp:= SearchResult{
			Type: "playlist",
			URI: &pls[index].URI,
			Name: pls[index].Name,
		}
		temp.Play()
		return
	}
	return
	}
	name:= concat(args)
	for _, p:= range pls{
		if strings.EqualFold(p.Name, name){
			temp:= SearchResult{
				Type: "playlist",
				Name: p.Name,
				URI: &p.URI,
			}
			temp.Play()
			return
		}
	}
	fmt.Printf("couldn't find any playlist of yours called %s\n", name)
}

func toggleShuffle(args []string) {
	if len(args)==0{
		err:= client.Shuffle(!shuffleState)
		if err!=nil{
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return
		}
		shuffleState= !shuffleState
		fmt.Printf("shuffle=%t\n", shuffleState)
		return
	}
	arg:= concat(args)
	switch strings.ToLower(arg){
		case "on", "true", "yes", "enable", "enabled", "1":
		err:= client.Shuffle(true)
		if err!=nil{
			fmt.Fprintf(os.Stderr, "err: %s\n", err)
			return
		}
		shuffleState= true
		fmt.Printf("shuffle=%t\n", shuffleState)
		case "no", "off", "disabled", "false", "0":
		err:= client.Shuffle(false)
		if err!=nil{
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return
		}
		shuffleState=false
		fmt.Printf("shuffle=%t\n", shuffleState)
	}
}