use crate::db::Statements;
use crate::error::Error;
use lazy_static::lazy_static;
use prometheus::{labels, opts, register_counter, register_gauge, register_histogram_vec};
use prometheus::{Counter, Gauge, HistogramVec};
use tokio_postgres::Client;
use tonic::{Request, Response, Status, Streaming};

lazy_static! {
    static ref PUSH_TRADE_C: Counter = register_counter!(opts!(
        "push_trade_grpc_requests_total",
        "Number of HTTP requests made.",
        labels! {"handler" => "all",}
    ))
    .unwrap();
    static ref HTTP_BODY_GAUGE: Gauge = register_gauge!(opts!(
        "example_http_response_size_bytes",
        "The HTTP response sizes in bytes.",
        labels! {"handler" => "all",}
    ))
    .unwrap();
    static ref PUSH_TRADE_H: HistogramVec = register_histogram_vec!(
        "push_trade_grpc_request_duration_seconds",
        "The HTTP request latencies in seconds.",
        &["handler"]
    )
    .unwrap();
}

pub mod pb {
    tonic::include_proto!("sync");
}

pub use pb::sync_service_server::SyncServiceServer;

type SyncMarketsRequest = Request<Streaming<pb::SyncMarketsRequest>>;
type TradeRequest = Request<pb::TradeRequest>;
type Reply = Result<Response<pb::Empty>, Status>;

#[derive(Debug)]
pub struct Service {
    pub client: Client,
    pub statements: Statements,
}

#[tonic::async_trait]
impl pb::sync_service_server::SyncService for Service {
    #[tracing::instrument]
    async fn sync_markets(&self, _request: SyncMarketsRequest) -> Reply {
        // let mut stream = request.into_inner();

        // while let Some(market) = stream.next().await {
        //     save_market(&self.pool, market?).await?;
        // }

        Ok(Response::new(pb::Empty {}))
    }

    #[tracing::instrument(skip(self))]
    async fn push_trade(&self, request: TradeRequest) -> Reply {
        tracing::trace!("Saving new trade information...");
        PUSH_TRADE_C.inc();

        let timer = PUSH_TRADE_H.with_label_values(&["all"]).start_timer();
        save_trade(&self, request.into_inner()).await?;
        timer.observe_duration();

        Ok(Response::new(pb::Empty {}))
    }
}

// async fn save_market(pool: &sqlx::PgPool, market: pb::SyncMarketsRequest) -> Result<(), Error> {
//     let query = sqlx::query_file!(
//         "sql/markets/sync.sql",
//         market.symbol,
//         market.status,
//         market.base_asset,
//         market.base_asset_precision,
//         market.quote_asset,
//         market.quote_asset_precision,
//         market.exchange,
//     );
//     query.execute(pool).await?;
//     Ok(())
// }

async fn save_trade(svc: &Service, trade: pb::TradeRequest) -> Result<(), Error> {
    let params: &[&(dyn tokio_postgres::types::ToSql + Sync)] = &[
        &trade.symbol,
        &trade.price,
        &trade.quantity,
        &trade.trade_time.to_string(),
        &trade.exchange,
    ];
    svc.client
        .execute(&svc.statements.insert_trade, params)
        .await?;
    Ok(())
}
