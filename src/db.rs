use tokio_postgres::{connect, Client, NoTls, Statement};

pub struct Statements {
    pub insert_trade: Statement,
}

impl std::fmt::Debug for Statements {
    fn fmt(&self, _: &mut std::fmt::Formatter<'_>) -> std::result::Result<(), std::fmt::Error> {
        Ok(())
    }
}

pub async fn init() -> (Client, Statements) {
    let url = std::env::var("DATABASE_URL").unwrap();
    let (client, connection) = connect(&url, NoTls).await.unwrap();

    tokio::spawn(async move {
        if let Err(e) = connection.await {
            eprintln!("connection error: {}", e);
        }
    });

    let insert_trade = client
        .prepare(std::include_str!("../sql/trades/insert.sql"))
        .await
        .unwrap();

    let statements = Statements { insert_trade };

    (client, statements)
}
