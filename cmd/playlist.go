package cmd

import (
	"fmt"
	"github.com/zmb3/spotify"
	"os"
	"strings"
)

func getPlaylists() ([]*Playlist, error) {
	plPage, err := client.CurrentUsersPlaylists()
	if err != nil {
		return nil, err
	}
	if len(plPage.Playlists) == 0 {
		return nil, fmt.Errorf("no playlist found")
	}
	var pl []*Playlist
	for _, p := range plPage.Playlists {
		pl = append(pl, &Playlist{Name: p.Name, ID: p.ID.String()})
	}
	return pl, nil
}

func listPlaylists() {
	pls, err := getPlaylists()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	for i, p := range pls {
		fmt.Printf("%d- %s\n", i, p.Name)
	}
}

func showPlaylist(args []string) {
	if len(args) == 0 {
		if selectedSimple == nil {
			fmt.Println("you must select a playlist first")
			return
		}
		results, err := client.GetPlaylistTracks(selectedSimple.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error fetching the tracks for %s: %s\n", selectedSimple.Name, err)
			return
		}
		if len(results.Tracks) == 0 {
			fmt.Printf("%s has no tracks in it\n", selectedSimple.Name)
			return
		}
		for i, t := range results.Tracks {
			if i == 40 {
				fmt.Println("use `edit` to potentially see more tracks")
			}
			artists := ""
			for _, art := range t.Track.Artists {
				artists += art.Name + ", "
			}
			fmt.Printf("%d- %s by %s\n", i, t.Track.Name, artists)
		}
		fmt.Println("returning")
		return
	}
	name := concat(args)
	results, err := getPlaylists()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching playlists: %s\n", err)
		return
	}
	for _, pl := range results {
		if strings.EqualFold(pl.Name, name) {
			tracks, err := client.GetPlaylistTracks(spotify.ID(pl.ID))
			if err != nil {
				fmt.Fprintf(os.Stderr, "error fetching the tracks for %s: %s\n", pl.Name, err)
				return
			}
			if len(tracks.Tracks) == 0 {
				fmt.Printf("%s has no tracks in it\n", pl.Name)
				return
			}
			for i, t := range tracks.Tracks {
				if i == 40 {
					fmt.Println("to see more tracks, use `edit`")
					return
				}
				artists := ""
				for _, art := range t.Track.Artists {
					artists += art.Name + ", "
				}
				fmt.Printf("%d- %s by %s\n", i, t.Track.Name, artists)
			}
		}
	}
}

func deletePlaylist() {
	fmt.Println("not supported yet")
}

func getSimplePlaylists() ([]spotify.SimplePlaylist, error) {
	page, err := client.CurrentUsersPlaylists()
	if err != nil {
		return nil, err
	}
	if len(page.Playlists) == 0 {
		return nil, fmt.Errorf("no playlists found")
	}
	return page.Playlists, nil
}

func getPlaylist(name string) (*Playlist, error) {
	pls, err := getPlaylists()
	if err != nil {
		return nil, err
	}
	for _, pl := range pls {
		if strings.EqualFold(pl.Name, name) {
			return pl, nil
		}
	}
	return nil, fmt.Errorf("no playlist by the name %s found", name)
}
