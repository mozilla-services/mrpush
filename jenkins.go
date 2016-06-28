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
					c <- IRCMessage{line.Args[0], "usage: !build job gitref"}
				} else if len(build) == 3 {
					jobName := channels[line.Args[0]].Projects[build[1]]
					gitRef := build[2]

					go buildJob(jenkins, jobName, gitRef, line.Args[0], c)
					// We need to sleep for a little bit before we poll
					time.Sleep(10 * time.Second)
					go statusBuild(jenkins, jobName, true, line.Args[0], c)

				}
			case strings.HasPrefix(line.Args[1], "!status"):
				status := strings.Split(line.Args[1], " ")
				if len(status) == 2 {
					jobName := channels[line.Args[0]].Projects[status[1]]
					go statusBuild(jenkins, jobName, false, line.Args[0], c)
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

func statusBuild(jenkins *gojenkins.Jenkins, jobName string, poll bool, ircchannel string, c chan IRCMessage) {

	job, err := jenkins.GetJob(jobName)
	buildState := ""

	if err != nil {
		log.Printf("Unable to get job information: %s\n", err.Error())
		c <- IRCMessage{ircchannel, "Unable to get status"}
	} else {
		if poll {
			buildState = pollBuildState(jenkins, job)
		} else {
			buildState = getBuildState(jenkins, job)
		}

		switch {
		case buildState == "SUCCESS":
			c <- IRCMessage{ircchannel, fmt.Sprintf("Job %s succeeded", jobName)}
		case buildState == "FAILURE":
			c <- IRCMessage{ircchannel, fmt.Sprintf("Job %s failed", jobName)}
		case buildState == "BUILDING":
			c <- IRCMessage{ircchannel, fmt.Sprintf("Job %s is building", jobName)}
		}
	}
}

// result is SUCCESS, FAILURE, or "" when building
func getBuildState(jenkins *gojenkins.Jenkins, job gojenkins.Job) string {

	last, err := jenkins.GetLastBuild(job)

	if err != nil {
		log.Printf("Unable to get build information: %s\n", err.Error())
	} else if last.Result == "" {
		return "BUILDING"

	}
	return last.Result
}

// poll getBuildState every 10 seconds to watch for state change
func pollBuildState(jenkins *gojenkins.Jenkins, job gojenkins.Job) string {

	for {
		last := getBuildState(jenkins, job)

		if last != "BUILDING" {
			return last
		}
		time.Sleep(10 * time.Second)
	}
	return ""
}
