package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

var (
	reader            = bufio.NewReader(os.Stdin)
	terminator        = make(chan os.Signal, 1)
	PROMPT     string = "libman> "
)

func init() {
	PROMPT = LibmanConfig.Prompt
}

func prompt() string {
	ch := make(chan string, 0)
	go PromptChan(ch)
	for {
		select {
		case input := <-ch:
			return input
		case <-terminator:
			playerCleanup()
			os.Exit(0)
		}
	}
}

func PromptNormal() string {
	fmt.Print("> ")
	text, _ := reader.ReadString('\n')
	text = strings.Replace(text, "\r\n", "", -1)
	text = strings.Replace(text, "\n", "", -1)
	return text
}

func PromptChan(ch chan<- string) {
	fmt.Print(PROMPT)
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

func setPrompt(args ...string) {
	if len(args) == 0 {
		return
	}
	PROMPT = concat(args) + " "
}
