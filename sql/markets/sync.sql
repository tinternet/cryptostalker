INSERT INTO markets (
    "symbol",
    "status",
    "base_asset",
    "base_asset_precision",
    "quote_asset",
    "quote_asset_precision",
    "exchange"
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    (SELECT "id" FROM exchanges WHERE "key" = $7)
)
ON CONFLICT ("symbol", "exchange")
DO
    UPDATE SET "status" = $2, "base_asset_precision" = $4, "quote_asset_precision" = $6