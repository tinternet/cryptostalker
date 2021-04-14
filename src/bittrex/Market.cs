using System;

namespace bittrex
{
    public class Market
    {
        public string Symbol { get; set; }
        public string BaseCurrencySymbol { get; set; }
        public string quoteCurrencySymbol { get; set; }
        public string minTradeSize { get; set; }
        public int Precision { get; set; }
        public string Status { get; set; }
        public DateTime CreatedAt { get; set; }
        public string[] prohibitedIn { get; set; }
        public string[] AssociatedTermsOfService { get; set; }
        public string[] Tags { get; set; }
    }
}
