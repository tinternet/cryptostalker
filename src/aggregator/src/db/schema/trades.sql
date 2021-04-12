CREATE UNLOGGED TABLE IF NOT EXISTS trades (
    "symbol"     TEXT NOT NULL,
    "price"      TEXT NOT NULL,
    "quantity"   TEXT NOT NULL,
    "trade_time" TIMESTAMP NOT NULL,
    "exchange"   TEXT NOT NULL
);
