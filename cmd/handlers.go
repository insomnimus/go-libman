package cmd

import(
"regexp"
"strings"
"fmt"
)

var rVolume= regexp.MustCompile(`^[\-\+][0-9]+$`)

func parsePlayerCommand(s string) {
	
	fields:= strings.Fields(s)
	if len(fields)==0{
		toggle()
		fmt.Print("\n")
		return
	}
	if rVolume.MatchString(fields[0]){
		changeVolume(fields[0])
		return
	}
	
	switch strings.ToLower(fields[0]){
	case "play":
	playTrack(fields[1:])
	case "pause":
	pause()
	case ">", "next", "n":
	playNext()
	case "<", "prev", "previous":
	playPrev()
	case "add":
	saveCurrentlyPlaying(fields[1:])
	case "h", "help":
	playerHelp()
	case "mute":
	changeVolume("-100")
	case "what", "?":
	showCurrentlyPlaying()
	case "volume", "vol":
	volume(fields[1:])
	default:
	fmt.Printf("unknown command %q\n", fields[0])
	}
}

func playerHelp() {
	msg:= `commands:
	`
	fmt.Println(msg)
}