INSERT INTO trades (
    "symbol",
    "price",
    "quantity",
    "trade_time",
    "exchange"
) VALUES (
    $1,
    $2,
    $3,
    TO_TIMESTAMP($4),
    $5
);
