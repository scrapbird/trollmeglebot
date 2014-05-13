package main

import (
	"fmt"
	"container/list"
	"strings"

	irc "github.com/fluffle/goirc/client"
)

var (
	serverAddress = "130.95.13.18:6667"
	nick = "trollmegle"
	name = "trollmegle"
	channel = "#trollmegle"

	commandPrefix = "."

	players list.List
)

func main () {
	ircConn := irc.SimpleClient (nick)
	ircConn.EnableStateTracking ()

	// add callbacks
	ircConn.AddHandler (irc.CONNECTED, func (conn *irc.Conn, line *irc.Line) {
		conn.Join (channel)
	})

	// And a signal on disconnect
	quit := make(chan bool)
	ircConn.AddHandler(irc.DISCONNECTED, func(conn *irc.Conn, line *irc.Line) {
		quit <- true
	})

	ircConn.AddHandler ("JOIN", func (conn *irc.Conn, line *irc.Line) {
		fmt.Println (line.Raw)
		conn.Privmsg (line.Nick, "Welcome to trollmegle, to join the game please type '" + commandPrefix + "join'")
	});

	// part / quit callback
	pqCallback := func (conn *irc.Conn, line *irc.Line) {
		fmt.Println (line.Raw)
		playerElem := findPlayer (line.Nick)
		if playerElem != nil {
			players.Remove (playerElem)
			conn.Privmsg (channel, line.Nick + " has left the game")
		}
	}
	ircConn.AddHandler ("PART", pqCallback);
	ircConn.AddHandler ("QUIT", pqCallback);

	ircConn.AddHandler ("PRIVMSG", func (conn *irc.Conn, line *irc.Line) {
		fmt.Println (line.Raw)
		fmt.Println (line.Args)
		message := line.Args[1]

		if strings.HasPrefix (message, commandPrefix) {
			// process this message
			command := strings.TrimPrefix (message, commandPrefix)
			switch command {
			case "join":
				// add user to the game
				players.PushBack (line.Nick)
				ircConn.Privmsg (channel, line.Nick + " has joined the game")
			case "leave":
				// remove the user from the game
				playerElem := findPlayer (line.Nick)
				if playerElem != nil {
					players.Remove (playerElem)
					ircConn.Privmsg (channel, line.Nick + " has left the game")
				} else {
					ircConn.Privmsg (line.Nick, "You aren't in the game")
				}
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

	// wait for disconnect
	<-quit
	fmt.Println ("Disconnected from irc server")
}

func findPlayer (nick string) *list.Element {
	var next *list.Element
	for e := players.Front (); e != nil; e = next {
		next = e.Next ()
		if e.Value == nick {
			return e
		}
	}
	return nil
}

