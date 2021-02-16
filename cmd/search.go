package cmd

import (
	"fmt"
	"github.com/zmb3/spotify"
	"strconv"
	"strings"
)

func (sr SearchResult) StringBare() string {
	msg := ""
	switch strings.ToLower(sr.Type) {
	case "track", "song", "album":
		msg = fmt.Sprintf("%s by %s", sr.Name, sr.Artists)
	case "playlist":
		msg = fmt.Sprintf("%s | %s", sr.Name, sr.Owner)
	default:
		msg = sr.Name
	}
	return msg
}

func (srs *SearchResults) ChooseInteractiveBare() {
	if len(*srs) == 0 {
		return
	}
	for i, r := range *srs {
		switch strings.ToLower(r.Type) {
		case "track":
			fmt.Printf("#%-2d %-30s by %-30s\n",
				i, r.Name, strings.Join(r.Artists, ", "))
		case "playlist", "userplaylist":
			fmt.Printf("#%-2d %-30s by %-25s\n",
				i, r.Name, r.Owner)
		case "artist":
			fmt.Printf("#%-2d %-45s\n",
				i, r.Name)
		case "album":
			fmt.Printf("#%-2d %-40s by %-30s\n",
				i, r.Name, strings.Join(r.Artists, ", "))
		default:
			fmt.Printf("#%-2d %-45s\n", i, r.Name)
		}
	}
	var input string
	fmt.Printf("choose (0-%d), blank or -1 to cancel:\n", len(*srs))
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
		if index < 0 || index >= len(*srs) {
			fmt.Printf("invalid input, enter between 0-%d, blank or -1 to cancel\n", len(*srs)-1)
			continue
		}
		(*srs)[index].Play()
		return
	}
}

func search(arg string, sType spotify.SearchType) (SearchResults, error) {
	if arg == "" {
		return nil, fmt.Errorf("search term can't be empty")
	}
	query := ""
	if strings.Contains(arg, "::") {
		split := strings.Split(arg, "::")
		if len(split) == 2 {
			query = fmt.Sprintf("%s artist:%s",
				strings.TrimSpace(split[0]),
				strings.TrimSpace(split[1]))
		} else {
			query = arg
		}
	} else {
		query = arg
	}
	page, err := client.Search(query, sType)
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
	if page.Albums != nil && page.Albums.Albums != nil {
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

func playSall(arg string) {
	if arg == "" {
		fmt.Println("missing argument for search")
		return
	}
	results, err := search(arg, spotify.SearchTypeArtist|spotify.SearchTypeAlbum|spotify.SearchTypeTrack|spotify.SearchTypePlaylist)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	results.chooseInteractive()
}

func playStra(arg string) {
	results, err := search(arg, spotify.SearchTypeTrack)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	results.ChooseInteractiveBare()
}

func playSalb(arg string) {
	results, err := search(arg, spotify.SearchTypeAlbum)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	results.ChooseInteractiveBare()
}

func playSpla(arg string) {
	results, err := search(arg, spotify.SearchTypePlaylist)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	results.ChooseInteractiveBare()
}

func playSart(arg string) {
	results, err := search(arg, spotify.SearchTypeArtist)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	results.ChooseInteractiveBare()
}
