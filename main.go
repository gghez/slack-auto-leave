package main

import (
	"bufio"
	"flag"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/nlopes/slack"
)

func isMeInChannel(api *slack.Client, c slack.Channel) bool {
	log.Printf("Checking channel %s for leaving...", c.Name)
	for _, mid := range c.Members {
		u, err := api.GetUserInfo(mid)
		if err != nil {
			log.Printf("failed to retrieve user info :%s", err)
			return false
		}

		if u.Name == os.Getenv("SLACK_MYSELF") {
			return true
		}

	}
	return false
}

func getChannelsToLeave(api *slack.Client) ([]slack.Channel, error) {
	lfPath, ok := os.LookupEnv("SLACK_LEAVE_CHANNELS")
	if !ok {
		lfPath = ".leave"
	}
	log.Printf("uses leave channels file '%s'", lfPath)

	f, err := os.Open(lfPath)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	var toLeave []string
	for scanner.Scan() {
		toLeave = append(toLeave, scanner.Text())
		log.Printf("channel %s flagged to leave.", scanner.Text())
	}

	channels, err := api.GetChannels(true)
	if err != nil {
		return nil, err
	}

	log.Printf("%d channels found.", len(channels))

	var ctl []slack.Channel
	for _, c := range channels {
		for _, tl := range toLeave {
			if c.Name == tl {
				if isMeInChannel(api, c) {
					ctl = append(ctl, c)
				} else {
					log.Printf("Not in channel %s", c.Name)
				}
			}
		}
	}

	return ctl, nil
}

func main() {
	envFile := flag.String("envfile", ".env", "Environment variable file to load.")
	flag.Parse()
	log.Printf("loading env vars from %s", *envFile)

	err := godotenv.Load(*envFile)
	if err != nil {
		panic(err)
	}

	api := slack.New(os.Getenv("SLACK_APP_TOKEN"))

	channels, err := getChannelsToLeave(api)
	if err != nil {
		panic(err)
	}

	for _, c := range channels {
		log.Printf("%s ready to leave", c.Name)

		msgOptions := []slack.MsgOption{
			slack.MsgOptionText("Sorry I'm not supposed to be there. Bye :)", false),
			slack.MsgOptionAsUser(true),
		}
		mc, ts, ms, err := api.SendMessage(c.ID, msgOptions...)
		if err != nil {
			panic(err)
		}

		sc, err := api.GetChannelInfo(mc)
		if err != nil {
			panic(err)
		}

		tfloat, _ := strconv.ParseFloat(ts, 64)
		t := time.Unix(int64(math.Floor(tfloat)), 0)

		log.Printf("message '%s' sent to %s at %s.", ms, sc.Name, t)
		notInChannel, err := api.LeaveChannel(c.ID)
		if err != nil {
			panic(err)
		} else if notInChannel {
			log.Print("was not in that chan. weird, but good :)")
		} else {
			log.Print("channel leaved.")
		}
	}

	log.Print("Auto-leave process completed.")
}
