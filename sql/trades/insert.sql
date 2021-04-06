INSERT INTO trades (
    "symbol",
    "price",
    "quantity",
    "trade_time",
    "exchange"
)
VALUES (
    $1,
    $2,
    $3,
    TO_TIMESTAMP(CAST($4::TEXT AS BIGINT) / 1000),
    $5
);
