package main

import (
	"io/ioutil"
	"log"

	"strings"

	"fmt"

	"github.com/TrevorDev/go-finance"
	"github.com/nlopes/slack"
)

func getStockInfo(tickers []string) (map[string]map[string]string, error) {
	return finance.GetStockInfo(
		tickers,
		[]string{
			finance.Stock_Exchange,
			finance.Dividend_Yield,
			finance.Dividend_Per_Share,
			finance.Year_Range,
			finance.Volume,
			finance.Last_Trade_Price_Only,
			finance.Earnings_Per_Share,
			finance.Book_Value,
		},
	)
}

func getTickersFromMessage(msg string) []string {
	stocks := strings.Replace(msg, "quote ", "", 1)
	return strings.Split(stocks, " ")
}

func isQuoteRequest(msg string) bool {
	return strings.HasPrefix(msg, "quote")
}

func normalizeWarrenRequest(msg string) string {
	return strings.Replace(msg, "warrenbot ", "", 1)
}

func isWarrenRequest(msg string) bool {
	return strings.HasPrefix(msg, "warrenbot")
}

func main() {
	apiTokenContents, err := ioutil.ReadFile("./slacktoken.txt")
	if err != nil {
		log.Fatal(err)
	}
	apiToken := strings.TrimRight(string(apiTokenContents), "\n \t")

	log.Println(string(apiToken))
	api := slack.New(string(apiToken))

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	channels, err := api.GetChannels(true)
	if err != nil {
		log.Fatal(err)
	}
	channelNameById := make(map[string]string, len(channels))
	for _, channel := range channels {
		channelNameById[channel.ID] = channel.Name
	}

	for msg := range rtm.IncomingEvents {
		switch msg.Data.(type) {
		case *slack.MessageEvent:
			var (
				replyFn     func(msg string) error
				channelName string
				basicReply  bool
			)
			ev, _ := msg.Data.(*slack.MessageEvent)
			channelName, ok := channelNameById[ev.Channel]
			text := ev.Text
			if ok {
				// We're in a normal channel if the map access to channelNameById worked.
				if !isWarrenRequest(text) {
					break
				} else {
					if text == "warrenbot" {
						basicReply = true
					}
					text = normalizeWarrenRequest(text)
					log.Println("Normalized Warren Request: " + text)
				}
				replyFn = func(msg string) error {
					_, _, err := api.PostMessage(channelName, msg, slack.PostMessageParameters{AsUser: true})
					return err
				}
			} else {
				// The map access failed, so we're in a DM.
				channelName = ev.Channel
				replyFn = func(msg string) error {
					_, _, ch, err := api.OpenIMChannel(ev.User)
					if err != nil {
						log.Println(err)
					}
					_, _, err = api.PostMessage(ch, msg, slack.PostMessageParameters{AsUser: true})
					return err
				}
			}
			if basicReply {
				replyFn("What do you want, sucker?")
				break
			}
			if isQuoteRequest(text) {
				info, err := getStockInfo(getTickersFromMessage(text))
				if err != nil {
					if err = replyFn("Sorry, I can't get those quotes."); err != nil {
						log.Println(err)
					}
					break
				}
				if err = replyFn(fmt.Sprint(info)); err != nil {
					log.Println(err)
				}
			}
		case *slack.UnmarshallingErrorEvent:
			ev, _ := msg.Data.(*slack.UnmarshallingErrorEvent)
			log.Println(ev.Error())
		case *slack.ChannelCreatedEvent:
			ev, _ := msg.Data.(*slack.ChannelCreatedEvent)
			channelNameById[ev.Channel.Name] = ev.Channel.ID
		case *slack.ConnectedEvent:
			ev, _ := msg.Data.(*slack.ConnectedEvent)
			fmt.Println("Infos:", ev.Info)
			fmt.Println("Connection counter:", ev.ConnectionCount)
		case *slack.IMHistoryChangedEvent:
			ev, _ := msg.Data.(*slack.IMHistoryChangedEvent)
			log.Printf("Type: %s, Latest %s, Timestamp: %s\n", ev.Type, ev.Latest, ev.Timestamp)
		default:
			log.Printf("Received Event %s\n", msg.Type)
		}
	}
}
