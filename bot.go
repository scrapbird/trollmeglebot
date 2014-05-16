package main

import (
	"fmt"
	"container/list"
	"strings"
	"flag"

	omegle "github.com/simon-weber/gomegle"
	irc "github.com/fluffle/goirc/client"
)

var (
	serverAddress = "130.95.13.18:6667"
	nick = "trollmeglebot"
	aliceNick = "alice|omg"
	bobNick = "bob|omg"
	name = "trollmeglebot"
	channel = "#trollmegle"

	commandPrefix = "."

	players list.List

	multiIrc bool
	omegleInit bool
	omegleAlice, omegleBob *omegle.Session
	ircAlice, ircBob *irc.Conn
	ircConn *irc.Conn
	ircConns [](*irc.Conn)
)

func main () {
	// check flags for multi-connect
	mconnect := flag.Bool ("multi-connect", true, "use or not use multiple irc connections")
	flag.Parse ()
	multiIrc = *mconnect

	ircConn = irc.SimpleClient (nick)

	if multiIrc {
		ircConns = make([](*irc.Conn), 3)
		ircAlice = irc.SimpleClient (aliceNick)
		ircBob = irc.SimpleClient (bobNick)
		ircConns[0] = ircConn
		ircConns[1] = ircAlice
		ircConns[2] = ircBob
	} else {
		ircConns = make([](*irc.Conn), 1)
		ircConns[0] = ircConn
	}
	quit := make(chan bool)
	for _, iconn := range ircConns {
		iconn.EnableStateTracking ()
		// add callbacks
		iconn.AddHandler (irc.CONNECTED, func (conn *irc.Conn, line *irc.Line) {
			conn.Join (channel)
		})

		// And a signal on disconnect
		iconn.AddHandler(irc.DISCONNECTED, func(conn *irc.Conn, line *irc.Line) {
			quit <- true
		})
	}

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

	// TODO: add handler for ircBob and ircAlice so can privmsg them
	ircConn.AddHandler ("PRIVMSG", func (conn *irc.Conn, line *irc.Line) {
		fmt.Println (line.Raw)
		fmt.Println (line.Args)
		message := line.Args[1]

		if strings.HasPrefix (message, bobNick + ": ") {
			message := strings.TrimPrefix (message, bobNick + ": ")
			if omegleInit {
				omegleBob.Message (message)
			}
		} else if strings.HasPrefix (message, aliceNick + ": ") {
			message := strings.TrimPrefix (message, aliceNick + ": ")
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
	for _, iconn := range ircConns {
		err := iconn.Connect (serverAddress)
		if err != nil {
			fmt.Println ("Error connecting to irc network")
			fmt.Println (err.Error ())
			return
		}
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
				if(multiIrc) {
					ircAlice.Privmsg (channel, event.Value)
				} else {
					ircConn.Privmsg (channel, who + ": " + event.Value)
				}
			} else {
				omegleAlice.Message (event.Value)
				if(multiIrc) {
					ircBob.Privmsg (channel, event.Value)
				} else {
					ircConn.Privmsg (channel, who + ": " + event.Value)
				}
			}
			fmt.Println (who + ": " + event.Value)
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
