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
        v = grpc::start(database) => v?,
        v = metrics::start() => v?,
        v = interrupt::start() => v?,
    }

    Ok(())
}
