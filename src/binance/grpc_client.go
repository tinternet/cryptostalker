package main

import (
	context "context"
	"flag"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	grpc "google.golang.org/grpc"
)

var grpcAddr = flag.String("grpc", "127.0.0.1:50051", "grpc server endpoint")

type GrpcClient struct {
	channel           *grpc.ClientConn
	client            SyncServiceClient
	tradesProcessed   prometheus.Counter
	tradesSentSuccess prometheus.Counter
	tradesSentFailed  prometheus.Counter
	tradesSentQueued  prometheus.Gauge
}

func (c GrpcClient) Close() {
	c.channel.Close()
}

func (c GrpcClient) PushTrades(trade *TradeRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	c.tradesProcessed.Inc()
	c.tradesSentQueued.Inc()

	_, err := c.client.PushTrade(ctx, trade)

	c.tradesSentQueued.Dec()

	if err == nil {
		c.tradesSentSuccess.Inc()
	} else {
		c.tradesSentFailed.Inc()
	}
}

func ConnectGRPC() GrpcClient {
	log.Println("Connecting to GRPC server:", *grpcAddr)
	conn, err := grpc.Dial(*grpcAddr, grpc.WithInsecure(), grpc.WithBlock())

	if err != nil {
		log.Fatalln("GRPC connect:", err)
	}

	return GrpcClient{
		channel: conn,
		client:  NewSyncServiceClient(conn),
		tradesProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "binance_grpc_processed_trades_total",
			Help: "The total number of processed trades",
		}),
		tradesSentSuccess: promauto.NewCounter(prometheus.CounterOpts{
			Name: "binance_grpc_sent_trades_success",
			Help: "The total number of processed trades",
		}),
		tradesSentFailed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "binance_grpc_sent_trades_failed",
			Help: "The total number of processed trades",
		}),
		tradesSentQueued: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "binance_processed_trades_queued",
			Help: "The total number of processed trades",
		}),
	}
}
