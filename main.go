package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Krognol/go-wolfram"
	wit "github.com/christianrondeau/go-wit"
	"github.com/joho/godotenv"
	"github.com/nlopes/slack"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

const confidenceThreshold = 0.5

var (
	slackClient   *slack.Client
	witClient     *wit.Client
	wolframClient *wolfram.Client
)

func main() {
	fmt.Println("starting app")
	slackClient = slack.New(os.Getenv("SLACK_ACCESS_TOKEN"))
	witClient = wit.NewClient(os.Getenv("WIT_AI_SERVER_ACCESS_TOKEN"))
	wolframClient = &wolfram.Client{AppID: os.Getenv("WOLFARM_APP_ID")}
	rtm := slackClient.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			go handleMessage(ev)
		}
	}
}

func handleMessage(ev *slack.MessageEvent) {
	result, err := witClient.Message(ev.Msg.Text)
	if err != nil {
		log.Printf("unable to get wit.ai result: %v", err)
		return
	}

	var (
		topEntity    wit.MessageEntity
		topEntityKey string
	)

	for key, entityList := range result.Entities {
		for _, entity := range entityList {
			if entity.Confidence > confidenceThreshold && entity.Confidence > topEntity.Confidence {
				topEntity = entity
				topEntityKey = key
			}
		}
	}
	replyToUser(ev, topEntity, topEntityKey)
}

func replyToUser(ev *slack.MessageEvent, topEntity wit.MessageEntity, topEntityKey string) {
	switch topEntityKey {
	case "greetings":
		slackClient.PostMessage(ev.User, slack.MsgOptionText("hello username", false),
			slack.MsgOptionAsUser(true))
		return

	case "wolfram_search_query":
		res, err := wolframClient.GetSpokentAnswerQuery(topEntity.Value.(string), wolfram.Metric, 1000)
		if err == nil {
			slackClient.PostMessage(ev.User, slack.MsgOptionText(res, false),
				slack.MsgOptionAsUser(true))
			return
		}
	default:
		slackClient.PostMessage(ev.User, slack.MsgOptionText("oops...I don't know", false),
			slack.MsgOptionAsUser(true))
	}
}
