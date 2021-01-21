package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

var (
	reader     = bufio.NewReader(os.Stdin)
	terminator = make(chan os.Signal, 1)
)

func prompt() string {
	ch := make(chan string, 0)
	go PromptChan(ch)
	for {
		select {
		case input := <-ch:
			return input
		case <-terminator:
			switch COMMAND {
			case "player":
				playerCleanup()
			case "local":
				searchCleanup()
			case "live":
				liveCleanup()
			}
			os.Exit(0)
		}

	}
}

func PromptNormal() string {
	fmt.Print(">")
	text, _ := reader.ReadString('\n')
	text = strings.Replace(text, "\r\n", "", -1)
	text = strings.Replace(text, "\n", "", -1)
	return text
}

func PromptChan(ch chan<- string) {
	fmt.Print(">")
	text, _ := reader.ReadString('\n')
	text = strings.Replace(text, "\r\n", "", -1)
	ch <- text
}

func minutes(d int) string {
	return fmt.Sprintf("%s", time.Duration(d)*time.Millisecond)
}

func yesOrNo() bool {
	input := ""
	for {
		input = prompt()
		switch strings.ToLower(input) {
		case "yeah", "yes", "y", "yep", "okay", "ok":
			return true
		case "no", "nope", "n", "nah":
			return false
		default:
			fmt.Println("please enter yes or no (y/n)")
		}
	}
	return false
}

func parseSearchCommand(input string) {
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return
	}
	// separate if input is `$abc` (so it'll be `$ abc`)
	if input[0] == '$' && len(input) > 1 && input[1] != ' ' {
		temp := fields[0][1:]
		fields = append([]string{"$", temp}, fields[1:]...)
	}
	switch strings.ToLower(fields[0]) {
	case "s", "search", "$":
		if len(fields) == 1 {
			fmt.Println("missing argument for search: query\nreturning")
			return
		}
		searchTrack(concat(fields[1:]))
		return
	case "choose", "cache", "c", "change", "cac", "select":
		chooseCache(fields[1:])
	case "new", "create", "n":
		createCache(fields[1:])
	case "help", "h":
		searchHelp(fields[1:])
	case "d", "del", "remove", "delete":
		deleteCache(fields[1:])
	case "e", "edit":
		editCache(fields[1:])
	case "list", "l":
		showCache(fields[1:])
	default:
		fmt.Printf("unknown command %q\n", fields[0])
	}
}

func concat(args []string) string {
	if len(args) == 0 {
		return ""
	}
	x := ""
	for _, s := range args {
		x += " " + s
	}
	return x[1:]
}

func searchHelp(_ []string) {
	fmt.Println(`commands:
		#new [name]: create a new cache
		(n, create)
		name can be omitted for an interactive screen
		example:
		new
		create metal
		
		#list [cache name]: list the contents of a cache
		the name can be omitted to list the selected cache
		(l)
		example:
		list metal
		
		#choose [cache name]: choose a cache
		the name can be omitted to get an interactive screen
		(c, select, cache, cac)
		example:
		choose
		choose metal
		
		#search <track>[::artist]: search for a song
		song - artist or song::artist is accepted for adding the artist to the search
		(s, $)
		example:
		search fear of the dark
		s fear of the dark :: iron maiden
		$ fear of the dark - iron maiden
		
		#delete [cache name]: delete a cache
		name can be omitted to get an interactive screen
		(del, remove, rm)
		example:
		delete metal
		rm metal
		
		#edit [cache name]: edit a cache
		name can be omitted to get an interactive screen
		(e)
		example:
		edit
		e metal
		
		#help (h): display this screen
		`)
}

func beginSearchLoop() {
	for {
		parseSearchCommand(prompt())
	}
}
