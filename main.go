package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func raiseError(w http.ResponseWriter, err error) {
	log.Println(err)
	w.WriteHeader(http.StatusInternalServerError)
}

func main() {
	api := slack.New(os.Getenv("SLACK_BOT_TOKEN"))

	http.HandleFunc("/slack/events", func(w http.ResponseWriter, r *http.Request) {
		// Initialize slack.SecretsVerifier
		// log.Println("header:", r.Header)
		verifier, err := slack.NewSecretsVerifier(r.Header, os.Getenv("SLACK_SINGING_SECRET"))
		if err != nil {
			raiseError(w, err)
			return
		}
		// log.Println("verfier:", verifier)

		// Read raw request
		bodyReader := io.TeeReader(r.Body, &verifier)
		body, err := ioutil.ReadAll(bodyReader)
		if err != nil {
			raiseError(w, err)
			return
		}
		// log.Println("body:", body)

		// Verify Request
		if err := verifier.Ensure(); err != nil {
			raiseError(w, err)
			return
		}

		// Parse request body to slack event object
		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
		if err != nil {
			raiseError(w, err)
			return
		}

		// Deal with slack event for types
		switch eventsAPIEvent.Type {
		case slackevents.URLVerification: // if type is url verification
			var res *slackevents.ChallengeResponse
			if err := json.Unmarshal(body, &res); err != nil { // check event body is challenge response // Unmarshal push body contents to value of res pointer
				raiseError(w, err)
				return
			}
			w.Header().Set("Content-Type", "text/plain")              // set response header
			if _, err := w.Write([]byte(res.Challenge)); err != nil { // write response body and send? // push res contents to response
				raiseError(w, err)
				return
			}
		case slackevents.CallbackEvent:
			innerEvent := eventsAPIEvent.InnerEvent
			switch event := innerEvent.Data.(type) {
			case *slackevents.AppMentionEvent:
				message := strings.Split(event.Text, " ") // message is "<@USERID> message"
				if len(message) < 2 {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				command := message[1]
				switch command {
				case "ping":
					if _, _, err := api.PostMessage(event.Channel, slack.MsgOptionText("pong", false)); err != nil {
						raiseError(w, err)
						return
					}
				case "pong":
					if _, _, err := api.PostMessage(event.Channel, slack.MsgOptionText("ping", false)); err != nil {
						raiseError(w, err)
						return
					}
				case "where":
					channelInfo, err := api.GetChannelInfo(event.Channel)
					if err != nil {
						raiseError(w, err)
						return
					}
					if _, _, err := api.PostMessage(event.Channel, slack.MsgOptionText(channelInfo.Name, false)); err != nil {
						raiseError(w, err)
						return
					}
				case "channels":
					var channels string
					channelList, err := api.GetChannels(false)
					if err != nil {
						raiseError(w, err)
						return
					}
					for i := 0; i < len(channelList); i++ {
						channels += "#" + channelList[i].Name + " / "
					}
					if _, _, err := api.PostMessage(event.Channel, slack.MsgOptionText(channels, false)); err != nil {
						raiseError(w, err)
						return
					}
				}
			}
		}
	})

	log.Println("[INFO] Server listening")
	if err := http.ListenAndServe(":8080", nil); err != nil { // start listening // if error happend call log.Fatal(err)
		log.Fatal(err)
	}
}
