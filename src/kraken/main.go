package main

import (
	context "context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	grpc "google.golang.org/grpc"
)

var addr = flag.String("addr", "ws.kraken.com", "websocket server endpoint")
var grpcaddr = flag.String("grpc", "127.0.0.1:50051", "grpc server endpoint")

type Pair struct {
	Altname             string
	Wsname              string
	Aclass_base         string
	Base                string
	Aclass_quote        string
	Quote               string
	Lot                 interface{}
	Pair_decimals       interface{}
	Lot_decimals        interface{}
	Lot_multiplier      interface{}
	Leverage_buy        interface{}
	Leverage_sell       interface{}
	Fees                interface{}
	Fees_maker          interface{}
	Fee_volume_currency interface{}
	Margin_call         interface{}
	Margin_stop         interface{}
	Ordermin            interface{}
}

type AssetPairs struct {
	Rrr    []interface{}
	Result map[string]Pair
}

type Message struct {
	Event        string            `json:"event"`
	Pair         []string          `json:"pair"`
	Subscription map[string]string `json:"subscription"`
}

func main() {
	flag.Parse()

	pairs := loadPairs()
	conn := connectGRPC()
	fetchTrades(pairs, conn)
}

func connectGRPC() *grpc.ClientConn {
	log.Println("Connecting to GRPC server:", grpcaddr)
	conn, err := grpc.Dial(*grpcaddr, grpc.WithInsecure(), grpc.WithBlock())

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	return conn
}

func fetchTrades(pairs []string, conn *grpc.ClientConn) {
	client := NewSyncServiceClient(conn)

	for {
		u := url.URL{Scheme: "wss", Host: *addr, Path: ""}
		log.Println("Connecting to websocket server:", u.String())
		ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

		if err != nil {
			log.Println("Server error:", err)
			log.Println("Retrying connection after 5 seconds.")
			<-time.After(5 * time.Second)
			continue
		}

		log.Printf("Subscibing for trade events.")
		err = ws.WriteJSON(&Message{
			Event: "subscribe",
			Pair:  pairs,
			Subscription: map[string]string{
				"name": "trade",
			},
		})

		if err != nil {
			ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, ""))
			ws.Close()
			log.Println("Server error:", err)
			log.Println("Reconnecting after 5 seconds.")
			<-time.After(5 * time.Second)
			continue
		}

		for {
			var message interface{}
			if err := ws.ReadJSON(&message); err != nil {
				log.Println("read:", err)
				log.Println("Closing connection!")
				ws.Close()
				break
			} else {
				for _, trade := range parseTrades(message) {
					pushTrade(trade, client)
				}
			}
		}
	}
}

func loadPairs() []string {
	for {
		log.Println("Fetching assets from:", "https://api.kraken.com/0/public/AssetPairs")
		resp, err := http.Get("https://api.kraken.com/0/public/AssetPairs")
		if err != nil {
			log.Println("Server error:", err)
			log.Println("Waiting 5 seconds before retry...")
			<-time.After(5 * time.Second)
			continue
		}

		var pairsResponse AssetPairs
		err = json.NewDecoder(resp.Body).Decode(&pairsResponse)
		if err != nil {
			log.Println("Server error:", err)
			log.Println("Waiting 5 seconds before retry...")
			<-time.After(5 * time.Second)
			continue
		}

		var pairs = make([]string, 0)
		for _, pair := range pairsResponse.Result {
			if pair.Wsname != "" {
				pairs = append(pairs, pair.Wsname)
			}
		}

		return pairs
	}
}

func pushTrade(trade *TradeRequest, client SyncServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := client.PushTrade(ctx, trade)
	if err != nil {
		log.Printf("Could not push trade: %v", err)
	} else {
		// log.Println("Successfully pushed trade", trade)
	}
}

func parseTrades(message interface{}) []*TradeRequest {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Could not parse response:", r, message)
		}
	}()

	_, ok := message.(map[string]interface{})
	if ok {
		// Ignore other messages
		return make([]*TradeRequest, 0)
	}

	array := message.([]interface{})
	symbol := array[3].(string)
	trades := array[1].([]interface{})
	parsed := make([]*TradeRequest, 0)

	for _, trade := range trades {
		t := trade.([]interface{})
		price := t[0].(string)
		volume := t[1].(string)
		timestr := t[2].(string)
		time, _ := strconv.ParseFloat(timestr, 64)
		parsed = append(parsed, &TradeRequest{
			Price:     price,
			Quantity:  volume,
			TradeTime: time,
			Symbol:    symbol,
			Exchange:  "kraken",
		})
	}

	return parsed
}
