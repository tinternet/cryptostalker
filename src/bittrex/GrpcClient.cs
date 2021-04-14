using Grpc.Core;
using System;
using System.Linq;
using System.Threading.Tasks;

namespace bittrex
{
    public class GrpcClient
    {
        SyncService.SyncServiceClient _client;

        public GrpcClient()
        {
            var grpcAddress = Environment.GetEnvironmentVariable("GRPC_SERVER_ADDR") ?? "127.0.0.1:50051";
            var channel = new Channel(grpcAddress, ChannelCredentials.Insecure);
            _client = new SyncService.SyncServiceClient(channel);
        }

        public async void PushTrade(TradeEvent ev)
        {
            var requests = ev.Deltas.Select(delta => new TradeRequest
            {
                Exchange = "bittrex",
                Price = delta.Rate,
                Quantity = delta.Quantity,
                Symbol = ev.MarketSymbol,
                TradeTime = delta.ExecutedAt.ToUnixTimeMilliseconds() / 1000
            });

            await Task.WhenAll(requests.Select(async request => await _client.PushTradeAsync(request)));
        }
    }
}
