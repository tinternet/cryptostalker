using System;

namespace bittrex
{
    public class Delta
    {
        public string ID { get; set; }
        public DateTimeOffset ExecutedAt { get; set; }
        public string Quantity { get; set; }
        public string Rate { get; set; }
        public string TakerSide { get; set; }
    }

    public class TradeEvent
    {
        public long Sequence { get; set; }
        public string MarketSymbol { get; set; }
        public Delta[] Deltas { get; set; }
    }
}
