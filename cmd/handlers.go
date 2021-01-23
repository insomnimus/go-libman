package cmd

import (
	"fmt"
	"regexp"
	"strings"
)

var rVolume = regexp.MustCompile(`^[\-\+][0-9]+$`)

func parsePlayerCommand(s string) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		toggle()
		fmt.Print("\n")
		return
	}
	if rVolume.MatchString(fields[0]) {
		changeVolume(fields[0])
		return
	}

	switch strings.ToLower(fields[0]) {
	case "shuffle", "shuf":
		toggleShuffle(fields[1:])
	case "choose", "select":
		choosePlaylist(fields[1:])
	case "edit":
		editSelectedPlaylist()
	case "stra", "strack", "searchtrack":
		if len(fields) == 1 {
			fmt.Println("missing argument for search")
		}
		playStra(concat(fields[1:]))
	case "sart", "sartist", "searchartist":
		if len(fields) == 1 {
			fmt.Println("missing argument for search")
		}
		playSart(concat(fields[1:]))
	case "salb", "salbum", "searchalbum":
		if len(fields) == 1 {
			fmt.Println("missing argument for search")
		}
		playSalb(concat(fields[1:]))
	case "spla", "spl", "splaylist", "searchplaylist":
		if len(fields) == 1 {
			fmt.Println("missing argument for search")
		}
		playSpla(concat(fields[1:]))
	case "search", "s", "sall", "sal":
		if len(fields) == 1 {
			fmt.Println("missing argument for search")
			return
		}
		playSall(concat(fields[1:]))
	case ">", "next", "n":
		playNext()
	case "<", "prev", "previous":
		playPrev()
	case "add", "save", "saveto":
		saveCurrentlyPlaying(fields[1:])
	case "h", "help":
		playerHelp()
	case "mute":
		changeVolume("-100")
	case "what", "?", "current":
		showCurrentlyPlaying()
	case "playlists", "pl":
		playUserPlaylist(fields[1:])
	case "volume", "vol":
		volume(fields[1:])
	case "device":
		chooseDevice()
	case "create", "new":
		createPlaylist()
	case "rename":
		renamePlaylist(fields[1:])
	case "repeat", "rep":
		cycleRepeatState(fields[1:])
	case "recommend", "rec", "recom":
		recommend(fields[1:])
	case "show", "sh":
		show(fields[1:])
	default:
		fmt.Printf("unknown command %q\n", fields[0])
	}
}

func playerHelp() {
	msg := `
	you can enter blank to play/pause
	you can change the volume by doing - or + followed by a number
	you can play next/prev song with >/< or next/prev
	
	
	commands:
	#s/search <name>
	search for anything, then if you want to, play it
	#shuffle
	toggle shuffle
	
	#volume <percentage>
	set the volume
	
	#repeat|rep [off|track|context]
	cycle repeat states
	
	#what/current
	show currently playing
	
	#save/add [playlist name]
	save currently playing to a playlist
	
	#playlists/pl [playlist name]
	play one of your playlists
	
	#create/new
	create a new playlist
	
	#device
	choose a playback device
	
	#select/choose
	choose a playlist (this command is for editing, does not affect playback)
	
	#edit
	edit the selected playlist
	
	#rename old >> new
	rename a playlist
	
	#recommend|rec <playlist name>
	get recommendations based on a user playlist
	
	
	#salb, sart, stra, spla:
	search for albums, artists, tracks or playlists respectively
	
	#show|sh [playlist|recommendation]
	show a playlists or recommendations contents
	arguments can be shortened like "pl, rec"
	
	#mute
	mute
	`
	fmt.Println(msg)
}
