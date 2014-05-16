package main

import (
	"fmt"
	"container/list"
	"strings"

	omegle "github.com/simon-weber/gomegle"
	irc "github.com/fluffle/goirc/client"
)

var (
	serverAddress = "130.95.13.18:6667"
	nick = "trollmeglebot"
	name = "trollmeglebot"
	channel = "#trollmegle"

	commandPrefix = "."

	players list.List

	omegleInit bool
	omegleAlice, omegleBob *omegle.Session
	ircConn *irc.Conn
)

func main () {
	ircConn = irc.SimpleClient (nick)
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

		if strings.HasPrefix (message, "a: ") {
			message := strings.TrimPrefix (message, "a: ")
			if omegleInit {
				omegleBob.Message (message)
			}
		} else if strings.HasPrefix (message, "b: ") {
			message := strings.TrimPrefix (message, "b: ")
			if omegleInit {
				omegleAlice.Message (message)
			}
		} else if strings.HasPrefix (message, commandPrefix) {
			// process this message
			command := strings.TrimPrefix (message, commandPrefix)
			switch command {
			case "join":
				// check if the user isn't already in the game
				playerElim := findPlayer (line.Nick)
				if playerElim != nil {
					// the player is already in the game
					ircConn.Privmsg (line.Nick, "You are already in the game")
				} else {
					// add user to the game
					players.PushBack (line.Nick)
					ircConn.Privmsg (channel, line.Nick + " has joined the game")
				}
			case "leave":
				// remove the user from the game
				playerElem := findPlayer (line.Nick)
				if playerElem != nil {
					players.Remove (playerElem)
					ircConn.Privmsg (channel, line.Nick + " has left the game")
				} else {
					ircConn.Privmsg (line.Nick, "You aren't in the game")
				}
			case "start":
				// start 2 omegle conversations and wait for them both to be ready
				ircConn.Privmsg (channel, "Starting new conversations")
				go startOmegleSessions ()
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

func startOmegleSessions () {
	fmt.Println ("Starting omegle sessions")

	omegleAlice = omegle.NewSession ()
	omegleBob = omegle.NewSession ()

	// create events channels
	aliceEvents := make (chan *omegle.Event, 16)
	bobEvents := make (chan *omegle.Event, 16)

	// go listen for events
	go listenForEvents (aliceEvents, "Alice")
	go listenForEvents (bobEvents, "Bob")

	// attempt to connect
	fmt.Println ("Attempting to connect to alice")
	go func () {
		err := omegleAlice.Connect (aliceEvents)
		if err != nil {
			fmt.Println ("Error connecting on Alice")
			ircConn.Privmsg (channel, "Error connecting on Alice")
		}
	}()

	fmt.Println ("Attempting to connect to bob")
	go func () {
		err := omegleBob.Connect (bobEvents)
		if err != nil {
			fmt.Println ("Error connecting on Bob")
			ircConn.Privmsg (channel, "Error connecting on Bob")
		}
	}()

	omegleInit = true
}

func listenForEvents (events chan *omegle.Event, who string) {
	fmt.Println ("Listening for events concerning " + who)
	for event := range events {
		fmt.Println (event)
		fmt.Println ("event type: " + event.Kind)
		if event.Kind == "connected" {
			fmt.Println (who + " connected")
			ircConn.Privmsg (channel, who + " connected")
		} else if event.Kind == "gotMessage" {
			if who == "Alice" {
				omegleBob.Message (event.Value)
			} else {
				omegleAlice.Message (event.Value)
			}
			fmt.Println (who + ": " + event.Value)
			ircConn.Privmsg (channel, who + ": " + event.Value)
		} else if event.Kind == "strangerDisconnected" {
			ircConn.Privmsg (channel, who + " disconnected")
		}
	}
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

func rotatePlayers () {
	currentPlayer := players.Front ()
	if currentPlayer != nil {
		players.MoveToBack (currentPlayer)
	}
}
