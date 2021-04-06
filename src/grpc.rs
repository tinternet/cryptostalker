// use crate::exchanges;
use crate::db;
use crate::sync;
use tonic::transport::Server;

pub async fn server() {
    let (client, prepared_statements) = db::init().await;

    let sync = sync::Service {
        client: client,
        statements: prepared_statements,
    };

    let addr = std::env::var("GRPC_ADDR").unwrap().parse().unwrap();
    tracing::info!("GRPC server listening on {}", addr);

    Server::builder()
        // .add_service(exchanges::ExchangeServiceServer::new(admin))
        .add_service(sync::SyncServiceServer::new(sync))
        .serve(addr)
        .await
        .unwrap();
}
