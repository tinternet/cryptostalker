#[macro_export]
macro_rules! prepare_statement {
    ($client:ident, $name:literal) => {{
        tracing::debug!("Preparing pgsql statement script [query/{}.sql]", $name);
        let query = std::include_str!(concat!("query/", $name, ".sql"));
        $client.prepare(query).await?
    }};
}

#[macro_export]
macro_rules! load_schema {
    ($client:ident, $name:literal) => {{
        tracing::debug!("Loading pgsql schema script [schema/{}.sql]", $name);
        let query = std::include_str!(concat!("schema/", $name, ".sql"));
        $client.execute(query, &[]).await?;
    }};
}

#[macro_export]
macro_rules! database {
    (
        schema {
            $($schema_name:ident => $schema_file:literal,)*
        }
        query {
            $($query_name:ident => $query_file:literal,)*
        }
    ) => {
        pub struct Database {
            client: tokio_postgres::Client,
            $($query_name: tokio_postgres::Statement,)*
        }

        impl Database {
            pub async fn connect() -> Result<Self, Box<dyn std::error::Error>> {
                let url = std::env::var("DATABASE_URL")?;
                tracing::info!("Connecting to databse at {}", url);
                let (client, connection) = tokio_postgres::connect(&url, tokio_postgres::NoTls).await?;

                // Spawn task to run the connection
                // FIXME: maybe need to catch disconnections?
                tokio::spawn(async move {
                    connection.await.unwrap();
                });

                $(
                    load_schema!(client, $schema_file);
                )*

                // I know this will conflict with variables above. I also don't care.
                $(
                    let $query_name = prepare_statement!(client, $query_file);
                )*

                Ok(Self {
                    client: client,
                    $($query_name: $query_name,)*
                })
            }

            fn slice_iter<'a>(&self, s: &'a [&'a (dyn tokio_postgres::types::ToSql + Sync)]) -> impl ExactSizeIterator<Item = &'a dyn tokio_postgres::types::ToSql> + 'a {
                s.iter().map(|s| *s as _)
            }

            $(
                pub async fn $query_name(&self, params: &[&(dyn tokio_postgres::types::ToSql + Sync)]) -> Result<(), Box<dyn std::error::Error>> {
                    self.client.execute_raw(&self.$query_name, self.slice_iter(params)).await?;
                    Ok(())
                }
            )*
        }
    }
}
