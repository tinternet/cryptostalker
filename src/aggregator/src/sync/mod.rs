use crate::db::Database;
use prometheus::{opts, register_counter, register_histogram, Counter, Histogram};
use tonic::{async_trait, Request, Response, Status};

pub mod pb;
pub use pb::sync_service_server::SyncServiceServer;

pub struct SyncService {
    db: Database,
    incoming_trades_total: Counter,
    incoming_trades_success: Counter,
    incoming_trades_failed: Counter,
    incoming_trades_performance: Histogram,
}

impl SyncService {
    pub fn new(db: Database) -> Self {
        let incoming_trades_total = register_counter!(opts!(
            "aggregator_incoming_trades_total",
            "Number of HTTP requests made."
        ));
        let incoming_trades_success = register_counter!(opts!(
            "aggregator_incoming_trades_success",
            "Number of HTTP requests made."
        ));
        let incoming_trades_failed = register_counter!(opts!(
            "aggregator_incoming_trades_failed",
            "Number of HTTP requests made."
        ));
        let incoming_trades_performance = register_histogram!(
            "aggregator_incoming_trade_rpc_duration_seconds",
            "The HTTP request latencies in seconds."
        );

        Self {
            db: db,
            incoming_trades_total: incoming_trades_total.unwrap(),
            incoming_trades_failed: incoming_trades_failed.unwrap(),
            incoming_trades_success: incoming_trades_success.unwrap(),
            incoming_trades_performance: incoming_trades_performance.unwrap(),
        }
    }
}

type TradeRequest = Request<pb::TradeRequest>;

#[async_trait]
impl pb::sync_service_server::SyncService for SyncService {
    #[tracing::instrument(skip(self))]
    async fn push_trade(&self, request: TradeRequest) -> Result<Response<pb::Empty>, Status> {
        tracing::trace!("Saving new trade");
        self.incoming_trades_total.inc();
        let timer = self.incoming_trades_performance.start_timer();
        let trade = request.into_inner();
        let result = self
            .db
            .insert_trade(&[
                &trade.symbol,
                &trade.price,
                &trade.quantity,
                &trade.trade_time,
                &trade.exchange,
            ])
            .await;

        match result {
            Ok(_) => {
                timer.observe_duration();
                self.incoming_trades_success.inc();
                Ok(Response::new(pb::Empty {}))
            }
            Err(e) => {
                timer.observe_duration();
                self.incoming_trades_failed.inc();
                tracing::error!("Unable to save trade {}", e);
                Err(Status::internal(e.to_string()))
            }
        }
    }
}
