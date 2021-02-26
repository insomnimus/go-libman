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
		playerHelp(fields[1:]...)
	case "mute":
		changeVolume("-100")
	case "what", "?", "current":
		showCurrentlyPlaying()
	case "playlists", "pl", "playlist":
		playUserPlaylist(fields[1:])
	case "volume", "vol":
		volume(fields[1:])
	case "device", "dev":
		chooseDevice()
	case "create", "new":
		createPlaylist()
	case "rename":
		renamePlaylist(fields[1:])
	case "repeat", "rep":
		cycleRepeatState(fields[1:])
	case "recommend", "rec", "recom":
		recommend(fields[1:])
	case "ls":
		show(fields[1:])
	case "show", "sh":
		show(fields[1:])
	case "prompt":
		if len(fields) == 1 {
			fmt.Println("usage: 'prompt <text>'")
			return
		}
		setPrompt(fields[1:]...)
	case "remove", "rm":
		removeCurrentlyPlaying(fields[1:])
	case "del", "delete", "unfollow":
		deletePlaylist(fields[1:])
	default:
		fmt.Printf("unknown command %q\n", fields[0])
	}
}

func playerHelp(args ...string) {
	if len(args) == 0 {
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
	
	#remove/rm [from]
	remove the currently playing track from a playlist
	
	#playlists/pl [playlist name]
	play one of your playlists
	
	#create/new
	create a new playlist
	
	#delete/del
	delete a playlist
	
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
	
	use 'help <command>' to get more info about a command
	`
		fmt.Println(msg)
		return
	}
	arg := concat(args)
	p := fmt.Printf
	switch strings.ToLower(arg) {
	case "s", "search":
		p(`#search (s) <search term>
		search anything musical
		example:
		's nightwish
		you can also use the 'track::artist' syntax
		s sahara::nightwish`)
	case "pl", "playlist":
		p(`#playlist (pl) [one of your playlists]
		play one of your playlists
		omit the playlist name to choose interactively`)
	case "strack", "stra":
		p(`#strack (stra) <track name>
		search for a track by its name
		example:
		stra and plague flowers the kaleidoscope
		this command also supports 'track::artist' syntax
		stra and plague flowers the kaleidoscope::ne obliviscaris`)
	case "sartist", "sart":
		p(`#sartist (sart) <artist name>
		search for an artist by their name
		example:
		sart insomnium`)
	case "salbum", "salb":
		p(`#salbum (salb) <album name>
		search for an album by its name, supports 'album::artist' syntax
		examples:
		salb ocean soul
		salb winter's gate::insomnium`)
	case "spla", "splaylist":
		p(`#splaylist (spla) <playlist name>
		search for a public spotify playlist by its name
		examples:
		spla death metal
		spla this is amorphis`)
	case "shuffle", "shuf":
		p(`#shuffle (shuf) [on/off]
		toggle shoffle
		if called without any arguments, toggles the shuffle state
		otherwise sets the shuffle state to on or off
		examples:
		shuffle
		shuffle off`)
	case "choose", "select":
		p(`#choose (select)
		select one of your playlists to edit
		does not affect playback`)
	case "edit":
		p(`#edit
		edit the selected playlist`)
	case "show", "sh":
		p(`#show (sh) [rec|playlist name]
		if called with no arguments, shows currently playing track
		if called with arguments;
		if the argument is "rec", shows the most recent recommendations list
		if the argument is any of "playlist", "pl" or "playlists"; lists your playlists
		otherwise shows one of your playlists contents
		examples:
		
		show
		show rec
		show metal`)
	case "ls":
		p(`#ls
		currently is the same as 'show'`)
	case "add", "save":
		p(`#save (add) [playlist name]
		saves the currently playing track to the specified playlist
		if no playlist name is given, you get to choose interactively`)
	case "rep", "repeat":
		p(`#repeat (rep) [off/track/context]
		toggle repeat state`)
	case "vol", "volume":
		p(`#volume (vol) [percentage]
		set the volume
		if used without any argument, displays it instead`)
	case ">", "next":
		p(`#next (>)
		play the next track`)
	case "previous", "prev", "<":
		p(`#prev (<)
		play the previous track`)
	case "mute":
		p(`#mute
		mute the volume`)
	case "device", "dev":
		p(`#device (dev)
		choose the playback device`)
	case "create":
		p(`#create
	create a new playlist`)
	case "rename":
		p(`#rename old>>new
	rename one of your playlists`)
	case "rec", "recommend":
		p(`#recommend (rec) [playlist name]
	generate some recommendations based on the given playlist name (has to be one of your playlists)
	if used without any argument, displays the most recent recommendations list`)
	case "rm", "remove":
		p(`#remove (rm) [playlist name]
	remove currently playing track from a playlist
	if the playlist name is omitted, the last played playlist will be assumed
	this command does not prompt for confirmation`)
	case "del", "delete", "unfollow":
		p(`#delete (del) [playlist name]
	deletes/unfollows a playlist
	if the name is omitted, you will be promptted to choose from a list of your playlists`)
	default:
		p("unknown command %q", arg)
	}
	fmt.Print("\n")
}
