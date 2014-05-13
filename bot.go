package main

import (
	"fmt"
	"container/list"
	"strings"
	"github.com/scrapbird/go-ircevent"
)

var (
	serverAddress = "irc.uniirc.com:6667"
	nick = "trollmeglee"
	name = "trollmegle"
	channel = "#trollmegle"

	commandPrefix = "."

	players list.List
)

func main () {
	ircConn := irc.IRC (nick, name)
	ircConn.Debug = true

	// add callbacks
	ircConn.AddCallback("001", func(event *irc.Event) {
		ircConn.Join(channel)
	})

	ircConn.AddCallback("JOIN", func (event *irc.Event) {
		fmt.Println (event.Message ())
		ircConn.Privmsg (event.Nick, "Welcome to trollmegle, to join the game please type '" + commandPrefix + "join'")
	});

	ircConn.AddCallback("PRIVMSG", func (event *irc.Event) {
		fmt.Println (event.Message ())
		if strings.HasPrefix (event.Message (), commandPrefix) {
			// process this message
			command := strings.TrimPrefix (event.Message (), commandPrefix)
			switch command {
			case "join":
				// add user to the game
				players.PushBack (event.Nick)
			}
		}
	});

	// connect to the network
	fmt.Println ("Attempting to connect to ", serverAddress)
	err := ircConn.Connect (serverAddress)
	if err != nil {
		fmt.Println ("Error connecting to irc network")
		fmt.Println (err.Error ())
		return
	}
	fmt.Println ("Connected")

	stop := make (chan int)
	var hammerTime int
	hammerTime = <- stop
	fmt.Println (hammerTime)
}
