fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::compile_protos("proto/exchanges.proto")?;
    tonic_build::compile_protos("proto/sync.proto")?;
    Ok(())
}
