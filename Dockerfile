FROM rust as planner
WORKDIR /usr/src/cryptostalker
# We only pay the installation cost once, 
# it will be cached from the second build onwards
# To ensure a reproducible build consider pinning 
# the cargo-chef version with `--version X.X.X`
RUN cargo install cargo-chef 
COPY . .
RUN cargo chef prepare  --recipe-path recipe.json

FROM rust as cacher
WORKDIR /usr/src/cryptostalker
RUN cargo install cargo-chef
COPY --from=planner /usr/src/cryptostalker/recipe.json recipe.json
RUN cargo chef cook --release --recipe-path recipe.json

FROM rust as builder
WORKDIR /usr/src/cryptostalker
COPY . .
# Copy over the cached dependencies
COPY --from=cacher /usr/src/cryptostalker/target target
COPY --from=cacher /usr/local/cargo /usr/local/cargo
RUN rustup component add rustfmt
RUN cargo build --release --bin cryptostalker

FROM rust as runtime
WORKDIR /usr/src/cryptostalker
COPY --from=builder /usr/src/cryptostalker/target/release/cryptostalker /usr/local/bin
ENTRYPOINT ["/usr/local/bin/cryptostalker"]
