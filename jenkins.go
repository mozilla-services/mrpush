package main

import (
	"fmt"
	irc "github.com/fluffle/goirc/client"
	"github.com/yosida95/golang-jenkins"
	"log"
	"net/url"
	"strings"
	"time"
)

func jenkinsActions(bot *irc.Conn, channels map[string]IRCChannels, j JenkinsCreds) {

	auth := &gojenkins.Auth{
		Username: j.Username,
		ApiToken: j.ApiToken,
	}

	jenkins := gojenkins.NewJenkins(auth, j.BaseUrl)

	c := make(chan IRCMessage)

	go sendMsg(bot, c)

	// line.Args[0] contains the channel/sender
	// line.Args[1] contains the message
	bot.HandleFunc(irc.PRIVMSG,
		func(conn *irc.Conn, line *irc.Line) {
			switch {
			case strings.HasPrefix(line.Args[1], "!list"):
				c <- IRCMessage{line.Args[0], "available jobs:"}
				for key, _ := range channels[line.Args[0]].Projects {
					c <- IRCMessage{line.Args[0], key}
				}
			case strings.HasPrefix(line.Args[1], "!build"):
				build := strings.Split(line.Args[1], " ")
				if len(build) == 1 {
					c <- IRCMessage{line.Args[0], "usage: !build project gitref"}
				} else if len(build) == 3 {
					jobName := channels[line.Args[0]].Projects[build[1]]
					gitRef := build[2]

					go buildJob(jenkins, jobName, gitRef, line.Args[0], c)
					time.Sleep(10 * time.Second)
					go statusJob(jenkins, jobName, line.Args[0], c)

				}
			case strings.HasPrefix(line.Args[1], "!status"):
				status := strings.Split(line.Args[1], " ")
				if len(status) == 2 {
					jobName := channels[line.Args[0]].Projects[status[1]]
					go statusJob(jenkins, jobName, line.Args[0], c)
				}
			}
		})

}

func buildJob(jenkins *gojenkins.Jenkins, jobName string, gitRef string, ircchannel string, c chan IRCMessage) {

	params := url.Values{}
	params.Add("GitRef", gitRef)

	job, err := jenkins.GetJob(jobName)

	if err != nil {
		log.Printf("Unable to get job information: %s\n", err.Error())

	} else {
		jenkins.Build(job, params)
		c <- IRCMessage{ircchannel, fmt.Sprintf("Building %s %s", jobName, gitRef)}
	}
}

func statusJob(jenkins *gojenkins.Jenkins, jobName string, ircchannel string, c chan IRCMessage) {

	job, err := jenkins.GetJob(jobName)

	if err != nil {
		log.Printf("Unable to get job information: %s\n", err.Error())
		c <- IRCMessage{ircchannel, "Unable to get status"}
	} else {
		log.Println(job)
		jobState := pollJobState(jenkins, job)
		// log.Println(job.Color)
		// log.Println(getlast.Result)

		if jobState == true {
			c <- IRCMessage{ircchannel, fmt.Sprintf("Build %s succeeded", jobName)}
		} else {
			c <- IRCMessage{ircchannel, fmt.Sprintf("Build %s failed", jobName)}
		}
	}
}

func pollJobState(jenkins *gojenkins.Jenkins, job gojenkins.Job) bool {
	for {
		last, err := jenkins.GetLastBuild(job)
		log.Println(last)
		if err != nil {
			log.Printf("Unable to get job information: %s\n", err.Error())
		} else if last.Result == "SUCCESS" {
			return true

		} else if last.Result == "FAILURE" {
			return false
		}
		time.Sleep(10 * time.Second)
	}
	return false
}
