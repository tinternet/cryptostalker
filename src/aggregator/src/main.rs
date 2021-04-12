mod db;
mod grpc;
mod interrupt;
mod metrics;
mod sync;

#[tokio::main(flavor = "current_thread")]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    dotenv::dotenv().ok();
    tracing_subscriber::fmt()
        .with_env_filter(tracing_subscriber::EnvFilter::from_default_env())
        .init();

    let database = db::Database::connect().await?;

    tokio::select! {
        _ = grpc::start(database) => {},
        _ = metrics::start() => {},
        _ = interrupt::start() => {},
    }

    Ok(())
}
