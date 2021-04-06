use crate::error::Error;
use sqlx::PgPool;
use tonic::{Request, Response, Status};

pub mod pb {
    tonic::include_proto!("exchanges");
}

pub use pb::exchange_service_server::ExchangeServiceServer;

type ListExchangesRequest = Request<pb::ListExchangesRequest>;
type ListExchangesResponse = Result<Response<pb::ListExchangesResponse>, Status>;
type AddExchangeRequest = Request<pb::AddExchangeRequest>;
type AddExchangeResponse = Result<Response<pb::AddExchangeResponse>, Status>;

#[derive(Debug)]
pub struct Service {
    pub pool: PgPool,
}

#[tonic::async_trait]
impl pb::exchange_service_server::ExchangeService for Service {
    #[tracing::instrument]
    async fn list_exchanges(&self, _request: ListExchangesRequest) -> ListExchangesResponse {
        tracing::info!("Listing exchanges...");
        let records = fetch_exchanges(&self.pool).await?;
        let reply = pb::ListExchangesResponse { exchanges: records };
        Ok(Response::new(reply))
    }

    #[tracing::instrument]
    async fn add_exchange(&self, request: AddExchangeRequest) -> AddExchangeResponse {
        let message = request.into_inner();
        add_exchange(&self.pool, &message.name, &message.description).await?;
        let reply = pb::AddExchangeResponse {};
        Ok(Response::new(reply))
    }
}

async fn fetch_exchanges(pool: &sqlx::PgPool) -> Result<Vec<pb::Exchange>, Error> {
    let query = sqlx::query_file_as!(pb::Exchange, "sql/exchanges/list.sql");
    let records = query.fetch_all(pool).await?;
    Ok(records)
}

async fn add_exchange(pool: &sqlx::PgPool, name: &str, description: &str) -> Result<(), Error> {
    let uuid = uuid::Uuid::new_v4().to_string();
    let query = sqlx::query_file!("sql/exchanges/add.sql", name, description, uuid);
    query.execute(pool).await?;
    Ok(())
}
