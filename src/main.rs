mod error;
// mod exchanges;
mod db;
mod grpc;
mod metrics;
mod sync;

#[tokio::main(flavor = "current_thread")]
// #[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    dotenv::dotenv().ok();
    tracing_subscriber::fmt()
        .pretty()
        .with_level(true)
        .with_ansi(true)
        .with_target(true)
        .with_env_filter(tracing_subscriber::EnvFilter::from_default_env())
        .init();

    tokio::join!(metrics::server(), grpc::server());

    Ok(())
}
