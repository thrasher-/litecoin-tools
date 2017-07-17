package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	SlackURL = "https://slack.com/api/rtm.start"
)

type Slack struct {
	Details       SlackResponse
	ReconnectURL  string
	WebsocketConn *websocket.Conn
	Connected     bool
	Channel       string
}

func (s Slack) BuildURL(token string) string {
	return fmt.Sprintf("%s?token=%s", SlackURL, token)
}

func (s Slack) GetChannelsString() []string {
	var channels []string
	for _, x := range s.Details.Channels {
		channels = append(channels, x.NameNormalized)
	}
	return channels
}

func (s Slack) GetChannelIDByName(channel string) string {
	for _, x := range s.Details.Channels {
		if x.Name == channel {
			return x.ID
		}
	}
	return ""
}

func (s Slack) GetUsernameByID(ID string) string {
	for _, x := range s.Details.Users {
		if x.ID == ID {
			return x.Name
		}
	}
	return ""
}

func (s Slack) GetGroupIDByName(group string) string {
	for _, x := range s.Details.Groups {
		if x.Name == group {
			return x.ID
		}
	}
	return ""
}

func (s Slack) GetUsersInGroup(group string) []string {
	for _, x := range s.Details.Groups {
		if x.Name == group {
			return x.Members
		}
	}
	return nil
}

func (s Slack) SendMessage(channel string, message string) error {
	var msg SlackSendMessage
	msg.ID = time.Now().Unix()
	msg.Type = "message"
	msg.Channel = channel
	msg.Text = message

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	s.WebsocketConn.WriteMessage(websocket.TextMessage, data)
	return nil
}

func SendHTTPGetRequestSlack(url string, jsonDecode bool, result interface{}) (err error) {
	res, err := http.Get(url)

	if err != nil {
		return
	}

	httpCode := res.StatusCode
	if httpCode != 200 && httpCode != 400 {
		log.Printf("HTTP status code: %d\n", httpCode)
		err = errors.New("Status code was not 200.")
		return
	}

	contents, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return
	}

	defer res.Body.Close()

	if jsonDecode {
		err := JSONDecode(contents, &result)

		if err != nil {
			return err
		}
	} else {
		result = string(contents)
	}

	return nil
}

func SlackConnect(token string, channelTarget string) {
	err := SendHTTPGetRequestSlack(slack.BuildURL(token), true, &slack.Details)
	if err != nil {
		log.Fatal(err)
	}

	if !slack.Details.Ok {
		log.Fatalf("Slack: Error: %s", slack.Details.Error)
	}

	log.Printf("%s [%s] connected to %s [%s] \nWebsocket URL: %s.\n", slack.Details.Self.Name, slack.Details.Self.ID, slack.Details.Team.Domain, slack.Details.Team.ID, slack.Details.URL)
	log.Printf("Slack channels: %s", slack.GetChannelsString())
	log.Printf("Channel target: %s ID: %s", channelTarget, slack.GetGroupIDByName(channelTarget))
	slack.Channel = slack.GetGroupIDByName(channelTarget)
	var Dialer websocket.Dialer

	for {
		url := slack.Details.URL
		if slack.ReconnectURL != "" {
			url = slack.ReconnectURL
		}
		slack.WebsocketConn, _, err = Dialer.Dial(url, http.Header{})

		if err != nil {
			log.Printf("Slack: Unable to connect to Websocket. Error: %s\n", err)
			time.Sleep(time.Second * 30)
		}

		for {
			_, resp, err := slack.WebsocketConn.ReadMessage()
			if err != nil {
				log.Println(err)
				break
			}

			type Response struct {
				Type    string `json:"type"`
				ReplyTo int    `json:"reply_to"`
			}

			var data Response
			err = JSONDecode(resp, &data)

			if err != nil {
				log.Println(err)
				continue
			}

			switch data.Type {
			case "hello":
				log.Println("Websocket connected successfully.")
				slack.Connected = true
			case "reconnect_url":
				type reconnectResponse struct {
					URL string `json:"url"`
				}
				var recURL reconnectResponse
				err = JSONDecode(resp, &recURL)
				if err != nil {
					continue
				}
				slack.ReconnectURL = recURL.URL
				log.Printf("Reconnect URL set to %s\n", slack.ReconnectURL)
			case "presence_change":
				var pres SlackPrescenseChange
				err = JSONDecode(resp, &pres)
				if err != nil {
					continue
				}
				log.Printf("Presence change. User %s [%s] changed status to %s\n", slack.GetUsernameByID(pres.User), pres.User, pres.Presence)
			case "message":
				if data.ReplyTo != 0 {
					continue
				}
				var msg SlackMessage
				err = JSONDecode(resp, &msg)
				if err != nil {
					continue
				}
				log.Printf("Msg received by %s [%s] with text: %s\n", slack.GetUsernameByID(msg.User), msg.User, msg.Text)
				slack.HandleMessage(msg)
			default:
				log.Println(string(resp))
			}
		}
	}
}

func (s Slack) HandleMessage(msg SlackMessage) {
	switch msg.Text {
	case "!status":
		if output.Status == "" {
			s.SendMessage(msg.Channel, "Bot is currently fetching data..")
			break
		}
		result := fmt.Sprintf("Status: %s, last updated: %d second(s) ago.", output.Status, GetSecondsElapsed(output.LastUpdated))
		s.SendMessage(msg.Channel, result)
	case "!hello":
		s.SendMessage(msg.Channel, fmt.Sprintf("Hello %s!", s.GetUsernameByID(msg.User)))
	case "!block":
		if output.Status == "" {
			s.SendMessage(msg.Channel, "Bot is currently fetching data..")
			break
		}
		info := fmt.Sprintf("Block height: %d Block time: %d Status: %s Seconds elapsed since last block: %d", output.Block.BlockHeight,
			output.Block.BlockTime, output.Block.Status, GetSecondsElapsed(output.Block.BlockTime))
		s.SendMessage(msg.Channel, info)
	}
}
