package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/zmb3/spotify"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	lastPl       *Playlist
	client       *spotify.Client
	repeatState  string = "off"
	activeDevice *spotify.PlayerDevice
	playerVolume = 80
	isPlaying    bool
	shuffleState bool
)

func StartPlayerSession() {
	log.SetFlags(0)
	log.SetPrefix("")
	rand.Seed(time.Now().UnixNano())
	signal.Notify(terminator, os.Interrupt)
	checkToken()
	initPlayer()
	for {
		parsePlayerCommand(prompt())
	}
}

func initPlayer() {
	activeDevice = getActiveDevice()
	if activeDevice != nil {
		playerVolume = activeDevice.Volume
		fmt.Printf("device: %s\n", activeDevice.Name)
	}
	plState, err := client.PlayerState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching player state: %s\n", err)
		os.Exit(2)
	}
	shuffleState = plState.ShuffleState
	repeatState = plState.RepeatState
	isPlaying = plState.Playing
	curPlaying := plState.Item
	if curPlaying != nil {
		artists := ""
		for _, art := range curPlaying.Artists {
			artists += art.Name + ", "
		}
		fmt.Printf("currently playing:\n%s by %s\n(pause=%t, volume= %d%%, shuffle=%t, repeat=%s)\n",
			curPlaying.Name,
			artists,
			!isPlaying,
			playerVolume,
			shuffleState,
			repeatState)
		return
	}
	fmt.Printf("shuffle=%t\n", shuffleState)

}

func saveCurrentlyPlaying(args []string) {
	playingTrack, err := currentlyPlayingTrack()
	if err != nil {
		fmt.Printf("failed to fetch currently playing info: %s\n", err)
		return
	}
	if len(args) == 0 {
		playlists, err := getPlaylists()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		for i, pl := range playlists {
			fmt.Printf("%d- %s\n", i, pl.Name)
		}
		fmt.Printf("add to which playlist? (0-%d), -1 or blank to cancel\n", len(playlists)-1)
		var input string
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
			if index < 0 || index >= len(playlists) {
				fmt.Printf("invalid input, enter 0-%d, blank or -1 to cancel:\n", len(playlists)-1)
				continue
			}
			playlists[index].addCache = append(playlists[index].addCache, playingTrack)
			playlists[index].Commit()
			fmt.Println("done")
			return
		}
	}
	name := concat(args)
	pls, err := getPlaylists()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	for _, p := range pls {
		if strings.EqualFold(p.Name, name) {
			p.addCache = append(p.addCache, playingTrack)
			p.Commit()
			fmt.Println("done")
			return
		}
	}
	fmt.Printf("no playlist found by the name %s\n", name)
}

func playPrev() {
	err := client.Previous()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}
	isPlaying = true
	fmt.Println("playing previous track")
}

func playNext() {
	err := client.Next()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}
	isPlaying = true
	fmt.Println("playing the next track")
}

func currentlyPlayingTrack() (Track, error) {
	var track Track
	cp, err := client.PlayerCurrentlyPlaying()
	if err != nil {
		return track, err
	}
	if !cp.Playing {
		return track, fmt.Errorf("not playing anything")
	}
	track.ID = string(cp.Item.ID)
	track.Duration = minutes(cp.Item.Duration)
	for _, art := range cp.Item.Artists {
		track.Artists = append(track.Artists, art.Name)
	}
	track.Name = cp.Item.Name
	return track, nil
}

func changeVolume(arg string) {
	amount, err := strconv.Atoi(arg)
	if err != nil {
		fmt.Printf("invalid input %s\n", arg)
		return
	}
	if playerVolume == 0 && amount <= 0 {
		fmt.Println("muted")
		return
	}
	if playerVolume == 100 && amount >= 0 {
		fmt.Println("max")
		return
	}
	playerVolume += amount
	if playerVolume < 0 {
		playerVolume = 0
	}
	if playerVolume > 100 {
		playerVolume = 100
	}
	err = client.Volume(playerVolume)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error setting the volume: %s\n", err)
		return
	}
}

func toggle() {
	var opt spotify.PlayOptions
	if activeDevice != nil {
		opt = spotify.PlayOptions{DeviceID: &activeDevice.ID}
	}
	if isPlaying {
		err := client.PauseOpt(&opt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
		}
		isPlaying = false
		return
	}
	err := client.PlayOpt(&opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
	}
	isPlaying = true
}

func volume(args []string) {
	if len(args) == 0 {
		fmt.Printf("the volume is %d%%\n", playerVolume)
		return
	}
	arg := concat(args)
	arg = strings.Replace(arg, " ", "", -1)
	if strings.Contains(arg, "+") || strings.Contains(arg, "-") {
		changeVolume(arg)
		return
	}
	amount, err := strconv.Atoi(arg)
	if err != nil {
		fmt.Printf("invalid argument for volume %q\n", arg)
		return
	}
	if amount <= 0 {
		changeVolume("-100")
		return
	}
	if amount >= 100 {
		changeVolume("+100")
		return
	}
	if amount == playerVolume {
		return
	}
	if amount > playerVolume {
		changeVolume(fmt.Sprintf("+%d", amount-playerVolume))
		return
	}

	changeVolume(fmt.Sprintf("-%d", playerVolume-amount))
}

func playerCleanup() {
	if db == nil {
		fmt.Println("fatal error: db is nil")
		os.Exit(1)
	}
	// when hit ctrl-c it switches play state, so do it again
	// only windows has this issue
	if os.PathSeparator == '\\' {
		toggle()
	}
	// save the token
	token, err := client.Token()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error saving token: %s\n", err)
		db.Close()
		os.Exit(1)
	}
	data, err := json.MarshalIndent(token, "", "\t")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling token: %s\n", err)
		db.Close()
		os.Exit(1)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("token"))
		err := b.Put([]byte("token"), data)
		return err
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error saving token to db: %s\n", err)
		db.Close()
		os.Exit(1)
	}
	db.Close()
	os.Exit(0)
}

func showCurrentlyPlaying() {
	tr, err := currentlyPlayingTrack()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Printf("currently playing %s\nshuffle=%t repeat=%s\n", tr, shuffleState, repeatState)
}

type SearchResult struct {
	Name, Owner string
	Artists     []string
	URI         *spotify.URI
	URIs        []spotify.URI
	Type        string
	ID          spotify.ID
}

type SearchResults []SearchResult

func (sr *SearchResult) Play() {
	refreshPlayer()
	defer refreshPlayer()
	var opt spotify.PlayOptions
	if activeDevice != nil {
		opt = spotify.PlayOptions{DeviceID: &activeDevice.ID}
	}
	switch strings.ToLower(sr.Type) {
	case "track":
		opt.URIs = sr.URIs
	case "userplaylist":
		opt.PlaybackContext = sr.URI
	case "album":
		var err error
		opt.URIs, err = collectAlbumURIs(sr.ID)
		if err != nil {
			fmt.Println("error:", err)
			return
		}
	case "playlist":
		var err error
		opt.URIs, err = collectPlaylistURIs(sr.ID)
		if err != nil {
			fmt.Println(err)
			return
		}
	case "artist":
		var err error
		opt.PlaybackContext, err = getArtistURI(sr.ID)
		if err != nil {
			fmt.Printf("error: %s\n", err)
			return
		}
	default:
		opt.PlaybackContext = sr.URI
	}
	err := client.PlayOpt(&opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}
	fmt.Printf("playing %s\n", sr.Name)
}

func collectAlbumURIs(id spotify.ID) ([]spotify.URI, error) {
	page, err := client.GetAlbumTracks(id)
	if err != nil {
		return nil, err
	}
	var uris []spotify.URI
	for _, t := range page.Tracks {
		uris = append(uris, t.URI)
	}
	return uris, nil
}

func getArtistURI(id spotify.ID) (*spotify.URI, error) {
	art, err := client.GetArtist(id)
	if err != nil {
		return nil, err
	}
	return &(art.URI), nil
}

func collectPlaylistURIs(id spotify.ID) ([]spotify.URI, error) {
	page, err := client.GetPlaylistTracks(id)
	if err != nil {
		return nil, err
	}
	if len(page.Tracks) == 0 {
		return nil, fmt.Errorf("no tracks could be fetched")
	}
	tracks := page.Tracks
	sort.Slice(tracks, func(i, j int) bool {
		return tracks[i].Track.Popularity > tracks[j].Track.Popularity
	})
	var uris []spotify.URI
	for _, track := range tracks {
		uris = append(uris, track.Track.URI)
	}
	return uris, nil
}

func (sr SearchResult) String() string {
	switch strings.ToLower(sr.Type) {
	case "track", "song":
		return fmt.Sprintf("track - %s by %s", sr.Name, strings.Join(sr.Artists, ", "))
	case "playlist":
		return fmt.Sprintf("playlist - %s | %s", sr.Name, sr.Owner)
	case "artist":
		return fmt.Sprintf("artist - %s", sr.Name)
	case "album":
		return fmt.Sprintf("album - %s by %s", sr.Name, strings.Join(sr.Artists, ", "))
	default:
		return fmt.Sprintf("result - %s", sr.Name)
	}
}

func (srs *SearchResults) Add(res ...SearchResult) {
	*srs = append(*srs, res...)
}

func (srs *SearchResults) chooseInteractive() {
	if len(*srs) == 0 {
		return
	}
	// to hold max 5 of each result type
	var displays SearchResults
	var art, alb, pla, tra SearchResults
	for _, res := range *srs {
		switch strings.ToLower(res.Type) {
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
	if len(tra) >= 5 {
		displays.Add(tra[:5]...)
	} else {
		displays.Add(tra...)
	}
	if len(pla) >= 5 {
		displays.Add(pla[:5]...)
	} else {
		displays.Add(pla...)
	}
	if len(art) >= 5 {
		displays.Add(art[:5]...)
	} else {
		displays.Add(art...)
	}
	if len(alb) >= 5 {
		displays.Add(alb[:5]...)
	} else {
		displays.Add(alb...)
	}
	for i, r := range displays {
		switch strings.ToLower(r.Type) {
		case "track":
			fmt.Printf("#%-2d %-10s %-20s by %-20s\n",
				i, "track", r.Name, strings.Join(r.Artists, ", "))
		case "playlist", "userplaylist":
			fmt.Printf("#%-2d %-10s %-20s by %-20s\n",
				i, "playlist", r.Name, r.Owner)
		case "artist":
			fmt.Printf("#%-2d %-10s %-45s\n",
				i, "artist", r.Name)
		case "album":
			fmt.Printf("#%-2d %-10s %-20s by %-20s\n",
				i, "album", r.Name, strings.Join(r.Artists, ", "))
		default:
			fmt.Printf("#%-2d %-10s %-45s\n", i, "-", r.Name)
		}
	}
	var input string
	fmt.Printf("choose (0-%d), blank or -1 to cancel:\n", len(displays))
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
		if index < 0 || index >= len(displays) {
			fmt.Printf("invalid input, enter between 0-%d, blank or -1 to cancel\n", len(displays)-1)
			continue
		}
		displays[index].Play()
		return
	}
}

// unused
func searchAll(arg string) (SearchResults, error) {
	if arg == "" {
		return nil, fmt.Errorf("missing argument `query` for search")
	}
	var query string
	if strings.Contains(arg, "::") {
		split := strings.Split(arg, "::")
		if len(split) == 2 {
			query = fmt.Sprintf("track:%s artist:%s",
				strings.TrimSpace(split[0]),
				strings.TrimSpace(split[1]))
		} else {
			query = arg
		}
	} else {
		query = arg
	}
	page, err := client.Search(query, spotify.SearchTypePlaylist|spotify.SearchTypeTrack|spotify.SearchTypeArtist|spotify.SearchTypeAlbum)
	if err != nil {
		return nil, err
	}
	var results SearchResults
	if page.Tracks != nil && len(page.Tracks.Tracks) > 0 {
		for _, t := range page.Tracks.Tracks {
			artists := make([]string, len(t.Artists))
			for i, art := range t.Artists {
				artists[i] = art.Name
			}
			results.Add(SearchResult{
				ID:      t.ID,
				Name:    t.Name,
				Artists: artists,
				Type:    "track",
				URIs:    []spotify.URI{t.URI},
			})
		}
	}
	if page.Playlists != nil && len(page.Playlists.Playlists) > 0 {
		for i, pl := range page.Playlists.Playlists {
			if i == 5 {
				break
			}
			owner, err := client.GetUsersPublicProfile(spotify.ID(pl.Owner.ID))
			if err != nil {
				continue
			}
			results.Add(SearchResult{
				Owner: owner.DisplayName,
				Name:  pl.Name,
				ID:    pl.ID,
				URI:   &(pl.URI),
				Type:  "playlist",
			})
		}
	}
	if page.Artists != nil && len(page.Artists.Artists) > 0 {
		for _, art := range page.Artists.Artists {
			results.Add(SearchResult{
				Name: art.Name,
				ID:   art.ID,
				URI:  &(art.URI),
				Type: "artist",
			})
		}
	}
	if page.Albums.Albums != nil && len(page.Albums.Albums) > 0 {
		for _, alb := range page.Albums.Albums {
			artists := make([]string, len(alb.Artists))
			for i, art := range alb.Artists {
				artists[i] = art.Name
			}
			results.Add(SearchResult{
				Name:    alb.Name,
				ID:      alb.ID,
				Artists: artists,
				URI:     &(alb.URI),
				Type:    "album",
			})
		}
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no result found for %s", query)
	}
	return results, nil
}

func playUserPlaylist(args []string) {
	page, err := client.CurrentUsersPlaylists()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching users playlists: %s\n", err)
		return
	}
	if len(page.Playlists) == 0 {
		fmt.Println("you don't seem to have any playlists")
		return
	}
	pls := page.Playlists

	if len(args) == 0 {
		fmt.Println(" #  | playlist")
		for i, p := range pls {
			fmt.Printf("#%-2d | %s\n", i, p.Name)
		}
		fmt.Printf("play which one? (0-%d), blank or -1 to cancel:\n", len(pls)-1)
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
			if index < 0 || index >= len(pls) {
				fmt.Printf("invalid input, enter 0-%d, blank or -1 to cancel:\n", len(pls)-1)
				continue
			}
			temp := SearchResult{
				Type: "userplaylist",
				URI:  &pls[index].URI,
				Name: pls[index].Name,
			}
			lastPl = &Playlist{
				Name: pls[index].Name,
				ID:   pls[index].ID.String(),
			}
			temp.Play()
			return
		}

	}
	name := concat(args)
	for _, p := range pls {
		if strings.EqualFold(p.Name, name) {
			temp := SearchResult{
				Type: "userplaylist",
				Name: p.Name,
				ID:   p.ID,
				URI:  &p.URI,
			}
			lastPl = &Playlist{
				Name: p.Name,
				ID:   p.ID.String(),
			}
			temp.Play()
			return
		}
	}
	fmt.Printf("couldn't find any playlist of yours called %s\n", name)
}

func toggleShuffle(args []string) {
	if len(args) == 0 {
		err := client.Shuffle(!shuffleState)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return
		}
		shuffleState = !shuffleState
		fmt.Printf("shuffle=%t\n", shuffleState)
		return
	}
	arg := concat(args)
	switch strings.ToLower(arg) {
	case "on", "true", "yes", "enable", "enabled", "1":
		err := client.Shuffle(true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "err: %s\n", err)
			return
		}
		shuffleState = true
		fmt.Printf("shuffle=%t\n", shuffleState)
	case "no", "off", "disabled", "false", "0":
		err := client.Shuffle(false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return
		}
		shuffleState = false
		fmt.Printf("shuffle=%t\n", shuffleState)
	}
}

func chooseDevice() {
	devices, err := client.PlayerDevices()
	if err != nil {
		fmt.Println(err)
		return
	}
	if len(devices) == 0 {
		fmt.Println("no device detected")
		return
	}
	fmt.Printf(" %-2s | %-20s | active\n", "no", "device name")
	for i, device := range devices {
		fmt.Printf("#%-2d | %-20s | %t\n", i, device.Name, device.Active)
	}
	fmt.Println("choose a device, enter blank or -1 to return")
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
		if index < 0 || index >= len(devices) {
			fmt.Printf("invalid input, enter 0-%d\n", len(devices)-1)
			continue
		}
		setDevice(devices[index])
		return
	}
}

func setDevice(device spotify.PlayerDevice) {
	if activeDevice != nil && device.ID == activeDevice.ID {
		fmt.Println("device already active")
		return
	}

	err := client.TransferPlayback(device.ID, isPlaying)
	if err != nil {
		fmt.Println(err)
		return
	}
	activeDevice = &device
	fmt.Printf("playing on %s\n", device.Name)
}

func getActiveDevice() *spotify.PlayerDevice {
	devices, err := client.PlayerDevices()
	if err != nil {
		fmt.Printf("error fetching the list of devices: %s\n", err)
		return nil
	}
	for _, device := range devices {
		if device.Active {
			return &device
		}
	}
	return nil
}

func cycleRepeatState(args []string) {
	refreshPlayer()
	if len(args) == 0 {
		switch repeatState {
		case "off":
			repeatState = "track"
		case "track":
			repeatState = "context"
		case "context":
			repeatState = "off"
		}
		err := client.Repeat(repeatState)
		if err != nil {
			fmt.Println("error: ", err)
			refreshPlayer()
			return
		}
		fmt.Printf("repeat=%s\n", repeatState)
		return
	}
	arg := strings.ToLower(concat(args))
	switch arg {
	case "off", "false":
		if repeatState == "off" {
			fmt.Println("repeat already off")
			return
		}
		repeatState = "off"
	case "track", "song", "on", "tr", "this":
		if repeatState == "track" {
			fmt.Println("repeat already set to track")
			return
		}
		repeatState = "track"
	case "playlist", "list", "context", "cont", "con":
		if repeatState == "context" {
			fmt.Println("repeat already set to context")
			return
		}
		repeatState = "context"
	default:
		fmt.Printf("unknown argument %q for repeat, valid arguments are 'track, context, off'\n", arg)
		return
	}
	err := client.Repeat(repeatState)
	if err != nil {
		fmt.Println("error:", err)
		refreshPlayer()
		return
	}
	fmt.Printf("repeat set to %s\n", repeatState)
}

func refreshPlayer() {
	devices, err := client.PlayerDevices()
	if err == nil {
		for _, device := range devices {
			if device.Active {
				playerVolume = device.Volume
				break
			}
		}
	}

	plState, err := client.PlayerState()
	if err == nil {
		shuffleState = plState.ShuffleState
		repeatState = plState.RepeatState
		isPlaying = plState.Playing
	}
}
