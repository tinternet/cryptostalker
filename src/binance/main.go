package main

import (
	context "context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	grpc "google.golang.org/grpc"
)

var binanceWs = flag.String("addr", "stream.binance.com:9443", "websocket server endpoint")
var binanceApi = flag.String("binance", "https://api.binance.com", "binance rest api")
var grpcaddr = flag.String("grpc", "127.0.0.1:50051", "grpc server endpoint")

type Symbol struct {
	Symbol                     string      `json:"symbol"`
	Status                     string      `json:"status"`
	BaseAsset                  string      `json:"baseAsset"`
	BaseAssetPrecision         int         `json:"baseAssetPrecision"`
	QuoteAsset                 string      `json:"quoteAsset"`
	QuotePrecision             int         `json:"quotePrecision"` // will be removed in future api versions (v4+)
	QuoteAssetPrecision        int         `json:"quoteAssetPrecision"`
	BaseCommissionPrecision    int         `json:"baseCommissionPrecision"`
	QuoteCommissionPrecision   int         `json:"quoteCommissionPrecision"`
	OrderTypes                 []string    `json:"orderTypes"`
	IcebergAllowed             bool        `json:"icebergAllowed"`
	OcoAllowed                 bool        `json:"ocoAllowed"`
	QuoteOrderQtyMarketAllowed bool        `json:"quoteOrderQtyMarketAllowed"`
	IsSpotTradingAllowed       bool        `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed     bool        `json:"isMarginTradingAllowed"`
	Filters                    interface{} `json:"filters"`
	Permissions                []string    `json:"permissions"`
}

type ExchangeInfo struct {
	Timezone        string      `json:"timezone"`
	ServerTime      int64       `json:"serverTime"`
	RateLimits      interface{} `json:"rateLimits"`
	ExchangeFilters interface{} `json:"exchangeFilters"`
	Symbols         []Symbol    `json:"symbols"`
}

type AggregatedTrade struct {
	EventType        string `json:"e"`
	EventTIme        int64  `json:"E"`
	Symbol           string `json:"s"`
	AggTradeID       int64  `json:"a"`
	Price            string `json:"p"`
	Quantity         string `json:"q"`
	FirstTradeID     int64  `json:"f"`
	LastTradeID      int64  `json:"l"`
	TradeTime        int64  `json:"T"`
	BuyerMarketMaker bool   `json:"m"`
	Ignore           bool   `json:"M"`
}

func chunkSlice(slice []string, chunkSize int) [][]string {
	var chunks [][]string
	for {
		if len(slice) == 0 {
			break
		}

		if len(slice) < chunkSize {
			chunkSize = len(slice)
		}

		chunks = append(chunks, slice[0:chunkSize])
		slice = slice[chunkSize:]
	}

	return chunks
}

func main() {
	info := exchangeInfo()
	conn := connectGRPC()
	pairs := tradeSubscriptions(info)
	fetchTrades(pairs, conn)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Block until a signal is received.
	<-interrupt
}

func exchangeInfo() ExchangeInfo {
	for {
		url := *binanceApi + "/api/v3/exchangeInfo"
		log.Println("Fetching assets from:", url)
		resp, err := http.Get(url)

		if err != nil {
			log.Println("Server error:", err)
			log.Println("Waiting 5 seconds before retry...")
			<-time.After(5 * time.Second)
			continue
		}

		var info ExchangeInfo
		err = json.NewDecoder(resp.Body).Decode(&info)

		if err != nil {
			log.Println("Server error:", err)
			log.Println("Waiting 5 seconds before retry...")
			<-time.After(5 * time.Second)
			continue
		}

		return info
	}
}

func connectGRPC() *grpc.ClientConn {
	log.Println("Connecting to GRPC server:", grpcaddr)
	conn, err := grpc.Dial(*grpcaddr, grpc.WithInsecure(), grpc.WithBlock())

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	return conn
}

type Subscription struct {
	ID     int      `json:"id"`
	Method string   `json:"method"`
	Params []string `json:"params"`
}

func tradeSubscriptions(info ExchangeInfo) []string {
	subscriptions := make([]string, 0)

	for _, symbol := range info.Symbols {
		value := strings.ToLower(symbol.Symbol) + "@aggTrade"
		subscriptions = append(subscriptions, value)
	}

	return subscriptions
}

func fetchTrades(pairs []string, conn *grpc.ClientConn) {
	client := NewSyncServiceClient(conn)
	connections := chunkSlice(pairs, 1024)

	for _, pairs := range connections {
		go connect(pairs, client)
	}
}

func connect(pairs []string, client SyncServiceClient) {
	for {
		basePair := pairs[0]
		u := url.URL{Scheme: "wss", Host: *binanceWs, Path: "/ws/" + basePair}
		log.Println("Connecting to websocket server:", u.String())
		ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

		if err != nil {
			log.Println("Server error:", err)
			log.Println("Retrying connection after 5 seconds.")
			<-time.After(5 * time.Second)

			continue
		}

		done := make(chan struct{}, 1)
		close := make(chan struct{}, 1)

		// Subscribe for trades in background
		go func() {
			if len(pairs) < 2 {
				return
			}

			index := 1
			subPairs := pairs[1:]
			subPairsChunks := chunkSlice(subPairs, 100)

			for _, chunk := range subPairsChunks {
				log.Println("Subscribing for pairs")

				err := ws.WriteJSON(Subscription{
					ID:     index,
					Method: "SUBSCRIBE",
					Params: chunk,
				})

				if err != nil {
					log.Println("suberror:", err)
					ws.Close()
					close <- struct{}{}
					return
				}

				<-time.After(time.Second * 2)
			}
		}()

	readLoop:
		for {
			select {
			case <-done:
				break readLoop
			default:
				var message AggregatedTrade
				if err := ws.ReadJSON(&message); err != nil {
					log.Println("read:", err)
					log.Println("Closing connection!")
					ws.Close()
					close <- struct{}{}
					break readLoop
				} else {
					if message.Symbol == "" {
						continue readLoop
					}
					go pushTrade(message, client)
				}
			}
		}
	}
}

func pushTrade(trade AggregatedTrade, client SyncServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*6)
	defer cancel()

	_, err := client.PushTrade(ctx, &TradeRequest{
		Price:     trade.Price,
		Quantity:  trade.Quantity,
		TradeTime: float64(trade.TradeTime) / 1000,
		Symbol:    trade.Symbol,
		Exchange:  "binance",
	})

	if err != nil {
		log.Printf("Could not push trade: %v", err)
	}
}
