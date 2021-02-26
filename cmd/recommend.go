package cmd

import (
	"fmt"
	"github.com/zmb3/spotify"
	"math/rand"
	"strings"
)

type Recommendation struct {
	Base   string
	Tracks []Track
}

func (rec *Recommendation) Add(t Track) {
	rec.Tracks = append(rec.Tracks, t)
}

func (rec *Recommendation) Play() {
	if len(rec.Tracks) == 0 {
		fmt.Println("recommendation does not have any tracks")
		return
	}
	refreshPlayer()
	defer refreshPlayer()
	opt := spotify.PlayOptions{DeviceID: &activeDevice.ID}
	for _, t := range rec.Tracks {
		opt.URIs = append(opt.URIs, t.URI)
	}
	err := client.PlayOpt(&opt)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("playing recommendations based on %s\n", rec.Base)
}

var recommendations *Recommendation

func recommend(args []string) {
	if len(args) == 0 && recommendations == nil {
		fmt.Println("first get some recommendations with `recommend <playlist name>`")
		return
	}
	if len(args) == 0 {
		recommendations.Play()
		return
	}
	name := concat(args)
	page, err := client.CurrentUsersPlaylists()
	if err != nil {
		fmt.Printf("recommend: %s\n", err)
		return
	}
	if len(page.Playlists) == 0 {
		fmt.Println("you don't seem to have any playlist")
		return
	}
	var plist *spotify.SimplePlaylist
	for _, pl := range page.Playlists {
		if strings.EqualFold(pl.Name, name) {
			plist = &pl
			break
		}
	}
	if plist == nil {
		fmt.Printf("no playlist found by the name %s\n", name)
		return
	}
	trPage, err := client.GetPlaylistTracks(plist.ID)
	if err != nil {
		fmt.Printf("recommend: %s\n", err)
		return
	}
	if len(trPage.Tracks) == 0 {
		fmt.Printf("playlist %s doesn't have any tracks yet\n", name)
		return
	}
	tempTracks := trPage.Tracks
	var tracks []spotify.FullTrack
	for i := 0; i < 5 && i < len(tempTracks); i++ {
		rn := rand.Intn(len(tempTracks))
		tracks = append(tracks, tempTracks[rn].Track)
		tempTracks = append(tempTracks[:rn], tempTracks[rn+1:]...)
	}
	var seeds spotify.Seeds
	for _, t := range tracks {
		rn := rand.Intn(8)
		if rn == 0 {
			seeds.Artists = append(seeds.Artists, t.Artists[0].ID)
		} else {
			seeds.Tracks = append(seeds.Tracks, t.ID)
		}
	}
	recom, err := client.GetRecommendations(seeds, nil, nil)
	if err != nil {
		fmt.Printf("recommend: %s\n", err)
		return
	}
	if recom == nil {
		fmt.Println("spotify returned nil")
		return
	}
	recommendations = &Recommendation{Base: name}
	for _, t := range recom.Tracks {
		artists := []string{}
		for _, art := range t.Artists {
			artists = append(artists, art.Name)
		}
		recommendations.Add(Track{
			ID:      string(t.ID),
			Name:    t.Name,
			Artists: artists,
			URI:     t.URI,
		})
	}
	recommendations.Show()
	fmt.Println("type `rec` or `recommend` to play")
}

func show(args []string) {
	if len(args) == 0 {
		showCurrentlyPlaying()
		return
	}
	name := concat(args)
	switch strings.ToLower(name) {
	case "rec", "recommend", "recommendation", "recommendations":
		if recommendations == nil {
			fmt.Println("first get some recommendations with `recommend <playlist name>`")
			return
		}
		recommendations.Show()
	case "pl", "playlists", "playlist":
		listPlaylists()
	default:
		showPlaylist(args)
	}
}

func (rec *Recommendation) Show() {
	fmt.Printf(" %-2s | %-25s | %25s\n", "no", "track", "artist")
	for i, t := range rec.Tracks {
		if i == 25 {
			break
		}
		fmt.Printf("#%-2d | %-25s | %25s\n", i, t.Name, strings.Join(t.Artists, ", "))
	}
	if len(rec.Tracks) > 25 {
		fmt.Printf("only showing 25 out of %d, though rest is also playable\n", len(rec.Tracks))
	}
}
