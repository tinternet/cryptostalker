package main

import (
	"strings"
	"time"
)

const MAX_STREAMS_PER_CONNECTION = 50

type BinanceStreamPool struct {
	events  chan StreamEvent
	streams []BinanceStream
}

func (pool BinanceStreamPool) Events() chan StreamEvent {
	return pool.events
}

func (pool BinanceStreamPool) Close() {
	for _, stream := range pool.streams {
		stream.Close()
	}
}

func CreateBinanceStreamPool(info ExchangeInfo) BinanceStreamPool {
	channels := make([]string, 0)

	for _, s := range info.Symbols {
		channel := strings.ToLower(s.Symbol) + "@aggTrade"
		channels = append(channels, channel)
	}

	// Need to span the channels into multiple connections
	// TODO: increase the streams per connection as Binance allows up to 1024
	channelChunks := make([][]string, 0)
	chunkSize := MAX_STREAMS_PER_CONNECTION

	for {
		if len(channels) == 0 {
			break
		}
		if len(channels) < chunkSize {
			chunkSize = len(channels)
		}
		channelChunks = append(channelChunks, channels[0:chunkSize])
		channels = channels[chunkSize:]
	}

	events := make(chan StreamEvent)
	streams := make([]BinanceStream, 0)

	go func() {
		for _, chunk := range channelChunks {
			stream := OpenBinanceStream(chunk)
			streams = append(streams, stream)

			go func() {
				for event := range stream.Events() {
					events <- event
				}
			}()

			// Wait for a while before opening another stream so Binance won't get upset
			<-time.After(time.Second)
		}
	}()

	return BinanceStreamPool{
		events:  events,
		streams: streams,
	}
}
