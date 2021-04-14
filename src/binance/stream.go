package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	binance_websocket_connections_open = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "binance_websocket_connections_open",
		Help: "The total number of processed trades",
	})
	binance_websocket_connection_reconnects_total = promauto.NewCounter(prometheus.CounterOpts{
		Name: "binance_websocket_connection_reconnects_total",
		Help: "The total number of processed trades",
	})
	binance_websocket_connection_errors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "binance_websocket_connection_errors",
		Help: "The total number of processed trades",
	})
	binance_websocket_streams_total = promauto.NewCounter(prometheus.CounterOpts{
		Name: "binance_websocket_streams_total",
		Help: "The total number of processed trades",
	})
	binance_websocket_streams_events_total = promauto.NewCounter(prometheus.CounterOpts{
		Name: "binance_websocket_streams_events_total",
		Help: "The total number of processed trades",
	})
)

// Contains aggregated trade information for single taker order
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

// Represents stream subscription event
type StreamEvent struct {
	Stream  string          `json:"stream"`
	Payload AggregatedTrade `json:"data"`
}

// Interface to Binance stream subscriptions
type BinanceStream struct {
	close    chan struct{}
	events   chan StreamEvent
	channels []string
}

// Channels returns list of subscribed channels
func (s BinanceStream) Channels() []string {
	return s.channels
}

// ReadEvent reads the next stream event from the Binance
func (s BinanceStream) Events() chan StreamEvent {
	return s.events
}

// Close closes the underlying websocket connection to Binance
func (s BinanceStream) Close() {
	s.close <- struct{}{}
}

// OpenBinanceStream connects to binance public websocket streams
func OpenBinanceStream(channels []string) BinanceStream {
	binance_websocket_streams_total.Add(float64(len(channels)))

	query := strings.Join(channels, "/")
	url := fmt.Sprintf("wss://stream.binance.com:9443/stream?streams=%v", query)

	events := make(chan StreamEvent)
	abort := make(chan struct{})

	sockets := make(chan *websocket.Conn)
	connect := make(chan struct{})
	stopReadLoop := make(chan struct{}, 1)

	go func() {
		defer close(events)
		defer close(sockets)
		defer close(connect)

		for {
			select {
			case <-connect:
				binance_websocket_connection_reconnects_total.Inc()
				sockets <- connectBinanceStream(url)
			case <-abort:
				stopReadLoop <- struct{}{}
				return
			}
		}
	}()

	go func() {
		for socket := range sockets {
			readBinanceStream(socket, events, stopReadLoop)
			connect <- struct{}{}
		}
	}()

	connect <- struct{}{}

	return BinanceStream{
		close:    abort,
		events:   events,
		channels: channels,
	}
}

func connectBinanceStream(url string) *websocket.Conn {
	for {
		log.Println("Connecting to binance stream at:", url)
		ws, _, err := websocket.DefaultDialer.Dial(url, nil)

		if err != nil {
			binance_websocket_connection_errors.Inc()
			log.Println("Binance stream error:", err)
			log.Println("Reconnecting to binance stream after 5 seconds")

			<-time.After(5 * time.Second)
			binance_websocket_connection_reconnects_total.Inc()
			continue
		}

		return ws
	}
}

func readBinanceStream(ws *websocket.Conn, events chan StreamEvent, abort chan struct{}) {
	// Be polite and perform gracefull disconnection
	defer ws.WriteMessage(websocket.CloseInternalServerErr, make([]byte, 0))
	defer ws.Close()

	binance_websocket_connections_open.Inc()
	defer binance_websocket_connections_open.Dec()

	for {
		select {
		case <-abort:
			return
		default:
		}

		var event StreamEvent
		err := ws.ReadJSON(&event)

		if err != nil {
			binance_websocket_connection_errors.Inc()
			log.Println("Binance stream read:", err)
			return
		}

		binance_websocket_streams_events_total.Inc()
		events <- event
	}
}
