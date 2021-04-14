package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var binanceApi = flag.String("binance", "https://api.binance.com", "binance rest api")

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

func main() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	}()

	exchangeInfo := fetchExchangeInfo(*binanceApi)

	grpcClient := ConnectGRPC()
	defer grpcClient.Close()

	pool := CreateBinanceStreamPool(exchangeInfo)
	defer pool.Close()

	for event := range pool.Events() {
		grpcClient.PushTrades(&TradeRequest{
			Price:     event.Payload.Price,
			Quantity:  event.Payload.Quantity,
			TradeTime: float64(event.Payload.TradeTime) / 1000,
			Symbol:    event.Payload.Symbol,
			Exchange:  "binance",
		})
	}
}

func fetchExchangeInfo(addr string) ExchangeInfo {
	url := *binanceApi + "/api/v3/exchangeInfo"
	log.Println("Fetching assets from:", url)
	resp, err := http.Get(url)

	if err != nil {
		log.Fatalln("Fetch exchanges:", err)
	}

	var info ExchangeInfo
	err = json.NewDecoder(resp.Body).Decode(&info)

	if err != nil {
		log.Fatalln("Parse exchanges:", err)
	}

	return info
}
