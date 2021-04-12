use crate::db::Database;
use crate::sync::{SyncService, SyncServiceServer};
use std::error::Error;
use tonic::transport::Server;

pub async fn start(db: Database) -> Result<(), Box<dyn Error>> {
    let addr = std::env::var("GRPC_LISTENER_ADDR")?.parse()?;
    tracing::info!("Grpc server listening on http://{}", addr);
    let sync = SyncServiceServer::new(SyncService::new(db));
    Server::builder().add_service(sync).serve(addr).await?;
    Ok(())
}
