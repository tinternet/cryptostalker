using Microsoft.AspNet.SignalR.Client;
using System;
using System.Collections.Generic;
using System.Security.Cryptography;
using System.Text;
using System.Threading.Tasks;
using System.Threading.Channels;

namespace bittrex
{
    public class SocketClient
    {
        private string _url;
        private HubConnection _hubConnection;
        private IHubProxy _hubProxy;

        public SocketClient(string url)
        {
            _url = url;
            _hubConnection = new HubConnection(_url);
            _hubProxy = _hubConnection.CreateHubProxy("c3");
        }

        public async Task<bool> Connect()
        {
            await _hubConnection.Start();
            return _hubConnection.State == ConnectionState.Connected;
        }

        public async Task<SocketResponse> Authenticate(string apiKey, string apiKeySecret)
        {
            var timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();
            var randomContent = $"{ Guid.NewGuid() }";
            var content = string.Join("", timestamp, randomContent);
            var signedContent = CreateSignature(apiKeySecret, content);
            var result = await _hubProxy.Invoke<SocketResponse>(
                "Authenticate",
                apiKey,
                timestamp,
                randomContent,
                signedContent);
            return result;
        }

        public IDisposable AddMessageHandler<Tmessage>(string messageName, Action<Tmessage> handler)
        {
            return _hubProxy.On(messageName, message =>
            {
                var decoded = DataConverter.DecodeCompressed<Tmessage>(message);
                handler(decoded);
            });
        }

        public void SetHeartbeatHandler(Action handler)
        {
            _hubProxy.On("heartbeat", handler);
        }

        public void SetAuthExpiringHandler(Action handler)
        {
            _hubProxy.On("authenticationExpiring", handler);
        }

        private static string CreateSignature(string apiSecret, string data)
        {
            var hmacSha512 = new HMACSHA512(Encoding.ASCII.GetBytes(apiSecret));
            var hash = hmacSha512.ComputeHash(Encoding.ASCII.GetBytes(data));
            return BitConverter.ToString(hash).Replace("-", string.Empty);
        }

        public async Task<List<SocketResponse>> Subscribe(string[] channels)
        {
            return await _hubProxy.Invoke<List<SocketResponse>>("Subscribe", (object)channels);
        }
    }
}
