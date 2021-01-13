package cmd

import (
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
		fmt.Println("play called")
	},
}

func init() {
	rootCmd.AddCommand(playerCmd)	
}

func startPlayerSession() {
	defer playerCleanup()
	signal.Notify(terminator, os.Interrupt)
	checkToken()
	initPlayer()
	fmt.Printf("welcome %s\n", user.DisplayName)
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
	track, err:= currentlyPlayingTrack()
	if err!=nil{
		fmt.Printf("failed to fetch currently playing info: %s\n", err)
		return
	}
	playingTrack:= Track{Name: track.Name, ID: track.ID}
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

func pause() {
	err:= client.Pause()
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}
	fmt.Println("paused")
}

func play() {
	err:= client.Play()
	if err!=nil{
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}
	fmt.Println("playing")
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
	if playerVolume==0{
		fmt.Println("muted")
		return
	}
	if playerVolume== 100{
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

func playTrack(args []string) {
	if len(args)== 0{
		play()
		return
	}
	// todo: search for song and play
	play()
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
	fmt.Printf("currently playing %s\n", tr)
}