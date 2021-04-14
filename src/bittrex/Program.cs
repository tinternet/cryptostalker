using System;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using System.Net.Http;


namespace bittrex
{
    class Program
    {
        const string MARKETS_URL = "https://api.bittrex.com/v3/markets";
        const string WEBSOCKET_URL = "https://socket-v3.bittrex.com/signalr";
        const string API_KEY = "";
        const string API_SECRET = "";

        static async Task Main(string[] args)
        {
            var markets = await FetchMarkets(MARKETS_URL);
            var grpc = new GrpcClient();
            var client = new SocketClient(WEBSOCKET_URL);

            if (await client.Connect())
            {
                Console.WriteLine("Connected to Bittrex WebSocket Server");
            }
            else
            {
                Console.WriteLine("Failed to connect to Bittrex WebSocket Server");
                return;
            }

            client.SetHeartbeatHandler(() => { /* noop */ });
            client.AddMessageHandler<TradeEvent>("trade", grpc.PushTrade);

            await Subscribe(client, markets);

            Console.ReadKey();
        }

        static async Task Subscribe(SocketClient client, Market[] markets)
        {
            var symbols = markets.Select(m => String.Format("trade_{0}", m.Symbol));
            var channels = new string[] { "heartbeat" }.Concat(symbols).ToArray();
            var response = await client.Subscribe(channels);
            for (int i = 0; i < channels.Count(); i++)
            {
                Console.WriteLine(response[i].Success ? $"{channels[i]}: Success" : $"{channels[i]}: {response[i].ErrorCode}");
            }
        }

        static async Task<Market[]> FetchMarkets(string url)
        {
            var client = new HttpClient();
            var response = await client.GetAsync(url);
            response.EnsureSuccessStatusCode();
            var responseBody = await response.Content.ReadAsStringAsync();
            return DataConverter.Decode<Market[]>(responseBody);
        }
    }
}
