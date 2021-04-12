#[macro_use]
mod macros;

database! {
    schema {
        trades => "trades",
    }
    query {
        insert_trade => "insert_trade",
    }
}
