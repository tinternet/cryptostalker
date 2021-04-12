using System;
using System.Net.WebSockets;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Collections.Generic;
using System.IO;
using System.IO.Compression;
using Grpc.Core;

namespace huobi
{
    public class MarketInfo
    {
        // [JsonInclude]
        [JsonPropertyName("status")]
        public string Status { get; set; }
        // [JsonInclude]
        [JsonPropertyName("data")]
        public IList<Market> Data { get; set; }
    }

    public class Market
    {
        // [JsonInclude]
        [JsonPropertyName("symbol")]
        public string Symbol { get; set; }
    }

    // {
    //   "ch": "market.btcusdt.trade.detail",
    //   "ts": 1489474082831, //system update time
    //   "tick": {
    //         "id": 14650745135,
    //         "ts": 1533265950234, //trade time
    //         "data": [
    //             {
    //                 "amount": 0.0099,
    //                 "ts": 1533265950234, //trade time
    //                 "id": 146507451359183894799,
    //                 "tradeId": 102043495674,
    //                 "price": 401.74,
    //                 "direction": "buy"
    //             }
    //             // more Trade Detail data here
    //         ]
    //   }
    // }

    public class Trade
    {
        // [JsonInclude]
        [JsonPropertyName("amount")]
        public decimal Amount { get; set; }
        // [JsonInclude]
        [JsonPropertyName("ts")]
        public long TradeTime { get; set; }
        // [JsonInclude]
        [JsonPropertyName("id")]
        public decimal ID { get; set; }
        // [JsonInclude]
        [JsonPropertyName("tradeId")]
        public long TradeID { get; set; }
        // [JsonInclude]
        [JsonPropertyName("price")]
        public decimal Price { get; set; }
        // [JsonInclude]
        [JsonPropertyName("direction")]
        public string Direction { get; set; }
    }

    public class Tick
    {
        // [JsonInclude]
        [JsonPropertyName("id")]
        public decimal ID { get; set; }
        // [JsonInclude]
        [JsonPropertyName("ts")]
        public long TradeTime { get; set; }
        // [JsonInclude]
        [JsonPropertyName("data")]
        public IList<Trade> Trades { get; set; }
    }

    public class IncomingMessage
    {
        // [JsonInclude]
        [JsonPropertyName("ping")]
        public Nullable<long> Ping { get; set; }

        // [JsonInclude]
        [JsonPropertyName("ch")]
        public string Channel { get; set; }

        // [JsonInclude]
        [JsonPropertyName("tick")]
        public Tick Tick { get; set; }
    }

    public class Pong
    {
        // [JsonInclude]
        [JsonPropertyName("pong")]
        public long Value { get; set; }
    }

    public class Subscription
    {
        // [JsonInclude]
        [JsonPropertyName("sub")]
        public string Topic { get; set; }

        // [JsonInclude]
        [JsonPropertyName("id")]
        public string ID { get; set; }
    }

    class Program
    {
        static readonly HttpClient client = new HttpClient();
        static readonly ClientWebSocket socket = new ClientWebSocket();

        static async Task Main()
        {
            var grpcAddress = Environment.GetEnvironmentVariable("GRPC_SERVER_ADDR") ?? "127.0.0.1:50051";
            Channel channel = new Channel(grpcAddress, ChannelCredentials.Insecure);
            var client = new SyncService.SyncServiceClient(channel);

            Console.WriteLine("Fetching market data");
            var markets = await FetchSymbols();

            Console.WriteLine("Connecting to websocket server");
            await ConnectWebsocket(markets, client);
        }


        static async Task<MarketInfo> FetchSymbols()
        {
            while (true)
            {
                try
                {
                    var response = await client.GetAsync("https://api.huobi.pro/v1/common/symbols");
                    response.EnsureSuccessStatusCode();
                    var responseBody = await response.Content.ReadAsStringAsync();
                    return JsonSerializer.Deserialize<MarketInfo>(responseBody);
                }
                catch (HttpRequestException e)
                {
                    Console.WriteLine("Error :{0} ", e.Message);
                    Console.WriteLine("Retrying after 5 seconds...");
                    await Task.Delay(1000 * 5, CancellationToken.None);
                }
            }
        }

        static async Task ConnectWebsocket(MarketInfo markets, SyncService.SyncServiceClient grpcclient)
        {
            while (true)
            {
                try
                {
                    var uri = new Uri("wss://api-aws.huobi.pro/ws");
                    await socket.ConnectAsync(uri, CancellationToken.None);
                    Task.WaitAll(Subscribe(markets), ReadWebsocket(grpcclient));
                }
                catch (WebSocketException e)
                {
                    Console.WriteLine("Error :{0} ", e.Message);
                    Console.WriteLine("Retrying after 5 seconds...");
                    await Task.Delay(1000 * 5, CancellationToken.None);
                }
            }
        }

        static async Task ReadWebsocket(SyncService.SyncServiceClient grpcclient)
        {
            Console.WriteLine("Reading from websocket...");
            var bufferSize = 1024 * 1000 * 1000;
            var buffer = new byte[bufferSize];
            while (true)
            {
                var segment = new ArraySegment<byte>(buffer, 0, bufferSize);
                var result = await socket.ReceiveAsync(segment, CancellationToken.None);
                if (result.EndOfMessage)
                {
                    var slice = new ArraySegment<byte>(buffer, 0, result.Count);
                    var message = "";
                    try
                    {
                        message = DecompressMessage(slice);
                    }
                    catch (Exception e)
                    {
                        Console.WriteLine("Ex: {0}", e.Message);
                    }
                    if (message != "")
                    {
                        await ProcessMessage(message, grpcclient);
                    }
                }
                else
                {
                    Console.WriteLine("Message too big. Closing connection.");
                    await socket.CloseAsync(WebSocketCloseStatus.MessageTooBig, null, CancellationToken.None);

                    Console.WriteLine("Reconnecting after 5 seconds...");
                    await Task.Delay(1000 * 5, CancellationToken.None);
                }
            }
        }

        static string DecompressMessage(ArraySegment<byte> data)
        {
            using (var compressedStream = new MemoryStream(data.Array))
            using (var zipStream = new GZipStream(compressedStream, CompressionMode.Decompress))
            using (var message = new MemoryStream())
            {
                zipStream.CopyTo(message);
                return System.Text.Encoding.Default.GetString(message.ToArray());
            }
        }

        static async Task ProcessMessage(string messageString, SyncService.SyncServiceClient grpcclient)
        {
            // Console.WriteLine("Received {0}", messageString);
            var message = JsonSerializer.Deserialize<IncomingMessage>(messageString);

            if (message.Ping != null)
            {
                // Console.WriteLine("Received ping. Responding back.", message);
                var pong = new Pong();
                pong.Value = message.Ping.Value;
                await socket.SendAsync(SerializeMessage(pong), WebSocketMessageType.Text, true, CancellationToken.None);
            }
            else if (message.Channel != null)
            {
                var symbol = message.Channel.Split(".")[1];
                foreach (var item in message.Tick.Trades)
                {
                    var req = new TradeRequest
                    {
                        Exchange = "huobi",
                        Price = item.Price.ToString(),
                        Quantity = item.Amount.ToString(),
                        Symbol = symbol,
                        TradeTime = item.TradeTime
                    };

                    try
                    {
                        var response = await grpcclient.PushTradeAsync(req);
                        // Console.WriteLine("Saved trade");
                    }
                    catch (Exception ex)
                    {
                        Console.WriteLine("Cannot send trade", ex.Message);
                    }
                }
            }
        }

        static ReadOnlyMemory<byte> SerializeMessage<T>(T message)
        {
            var str = JsonSerializer.Serialize<T>(message);
            var bytes = System.Text.Encoding.Default.GetBytes(str);
            return bytes.AsMemory();
        }

        static async Task Subscribe(MarketInfo info)
        {
            foreach (var asset in info.Data)
            {
                var sub = new Subscription();
                var topic = String.Format("market.{0}.trade.detail", asset.Symbol);
                sub.ID = topic;
                sub.Topic = topic;
                var msg = SerializeMessage(sub);
                await socket.SendAsync(msg, WebSocketMessageType.Text, true, CancellationToken.None);
            }
            // {
            //     "sub": "market.btcusdt.trade.detail",
            //     "id": "id1"
            //     }
        }
    }
}
