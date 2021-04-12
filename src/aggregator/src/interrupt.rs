use std::error::Error;
use tokio::signal::unix::{signal, SignalKind};

pub async fn start() -> Result<(), Box<dyn Error>> {
    let mut int = signal(SignalKind::interrupt())?;
    let mut term = signal(SignalKind::terminate())?;
    let mut quit = signal(SignalKind::quit())?;
    let mut hang = signal(SignalKind::hangup())?;

    tokio::select! {
        _ = tokio::signal::ctrl_c() => {
            println!("Exiting process due to ctrl-c");
        }
        _ = int.recv() => {
            println!("Exiting process due to SIGINT");
        }
        _ = term.recv() => {
            println!("Exiting process due to SIGTERM");
        }
        _ = quit.recv() => {
            println!("Exiting process due to SIGQUIT");
        }
        _ = hang.recv() => {
            println!("Exiting process due to SIGHUP");
        }
    }

    Ok(())
}
