#[derive(Debug)]
pub struct Error {
    message: String,
}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::result::Result<(), std::fmt::Error> {
        write!(f, "Error: {}", self.message)
    }
}

impl std::error::Error for Error {}

impl From<Error> for tonic::Status {
    fn from(error: Error) -> Self {
        tracing::error!("Returning error: {}", error.message);
        Self::internal(error.message)
    }
}

impl From<tokio_postgres::Error> for Error {
    fn from(error: tokio_postgres::Error) -> Self {
        tracing::error!("{}", error);
        Self {
            message: error.to_string(),
        }
    }
}
