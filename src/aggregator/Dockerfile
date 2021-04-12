FROM rust as planner
WORKDIR /usr/src/aggregator
# We only pay the installation cost once, 
# it will be cached from the second build onwards
# To ensure a reproducible build consider pinning 
# the cargo-chef version with `--version X.X.X`
RUN cargo install cargo-chef --version 0.1.19
COPY . .
RUN cargo chef prepare  --recipe-path recipe.json

FROM rust as cacher
WORKDIR /usr/src/aggregator
RUN cargo install cargo-chef
COPY --from=planner /usr/src/aggregator/recipe.json recipe.json
RUN cargo chef cook --release --recipe-path recipe.json

FROM rust as builder
WORKDIR /usr/src/aggregator
COPY . .
# Copy over the cached dependencies
COPY --from=cacher /usr/src/aggregator/target target
COPY --from=cacher /usr/local/cargo /usr/local/cargo
RUN rustup component add rustfmt
RUN cargo build --release --bin aggregator

FROM rust as runtime
WORKDIR /usr/src/aggregator
COPY --from=builder /usr/src/aggregator/target/release/aggregator /usr/local/bin
ENTRYPOINT ["/usr/local/bin/aggregator"]
