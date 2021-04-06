CREATE TABLE IF NOT EXISTS markets (
    "symbol"                TEXT NOT NULL,
    "status"                TEXT NOT NULL,
    "base_asset"            TEXT NOT NULL,
    "base_asset_precision"  INTEGER NOT NULL,
    "quote_asset"           TEXT NOT NULL,
    "quote_asset_precision" INTEGER NOT NULL,
    "exchange"              TEXT NOT NULL 
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_markets_symbol_exchange ON markets (symbol, exchange);
