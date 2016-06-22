package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	irc "github.com/fluffle/goirc/client"
	"log"
	// jenkins "github.com/yosida95/golang-jenkins"
	"os"
	"runtime"
	"strings"
)

type IRCChannels struct {
	Key      string
	Projects map[string]string
}

type IRCConfig struct {
	Nick     string
	Password string
	Host     string
	Ssl      bool
	Port     int
	Channels map[string]IRCChannels
}

func readConfig(configFile string) IRCConfig {
	config := IRCConfig{}

	file, _ := os.Open(configFile)
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&config)

	if err != nil {
		log.Println("Cannot parse config:", err)
		os.Exit(1)
	}

	return config
}

func Bot(config IRCConfig) {

	c := irc.NewConfig(config.Nick)
	c.SSL = config.Ssl
	c.SSLConfig = &tls.Config{ServerName: config.Host, InsecureSkipVerify: false}
	c.Server = config.Host
	c.NewNick = func(n string) string { return n + "^" }

	bot := irc.Client(c)

	quit := make(chan bool)

	bot.HandleFunc(irc.CONNECTED,
		func(conn *irc.Conn, line *irc.Line) {
			conn.Mode(conn.Me().Nick, "+B")
			bot.Privmsg("NickServ", fmt.Sprintf("identify %s", config.Password))
			for key, channel := range config.Channels {
				conn.Join(key + " " + channel.Key)
			}
		})

	bot.HandleFunc(irc.PRIVMSG,
		func(conn *irc.Conn, line *irc.Line) {
			// fmt.Println(line.Args[1])
			switch {
			case strings.HasPrefix(line.Args[1], "!build"):
				build := strings.Split(line.Args[1], " ")
				if len(build) == 1 {
					bot.Privmsg(line.Args[0], "usage: !build project gitref")
					bot.Privmsg(line.Args[0], "available projects:")
					for key, _ := range config.Channels[line.Args[0]].Projects {
						bot.Privmsg(line.Args[0], key)
					}
				} else if len(build) == 3 {
					a := config.Channels[line.Args[0]].Projects[build[1]]
					bot.Privmsg(line.Args[0], a)
				}
			case strings.HasPrefix(line.Args[1], "!quit"):
				quit <- true

			}
		})

	bot.HandleFunc(irc.DISCONNECTED,
		func(conn *irc.Conn, line *irc.Line) { quit <- true })

	if err := bot.Connect(); err != nil {
		log.Printf("Connection error: %s\n", err.Error())
	}

	<-quit

}

func main() {
	runtime.GOMAXPROCS(2)
	config := readConfig("config.json")
	// fmt.Println(config.Channels)
	Bot(config)
}
