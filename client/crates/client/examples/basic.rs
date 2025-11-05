use ev_client::{health::HealthClient, Client, P2PClient, StoreClient};
use std::error::Error;

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    // Initialize tracing for better debugging
    tracing_subscriber::fmt::init();

    // Connect to a Evolve node
    let endpoint =
        std::env::var("EVOLVE_ENDPOINT").unwrap_or_else(|_| "http://localhost:50051".to_string());
    println!("Connecting to evolve node at: {endpoint}");

    let client = Client::connect(&endpoint).await?;
    println!("Successfully connected to evolve node");

    // Check health status (HTTP endpoints)
    println!("\n=== Health Check ===");
    let health = HealthClient::with_base_url(endpoint.clone());

    // Liveness check - is the process alive?
    match health.livez().await {
        Ok(status) => {
            println!("Liveness status: {:?}", status);
            println!("Node is alive: {}", health.is_healthy().await?);
        }
        Err(e) => println!("Failed to get liveness status: {e}"),
    }

    // Readiness check - can it serve correct data?
    match health.readyz().await {
        Ok(status) => {
            println!("Readiness status: {:?}", status);
            println!("Node is ready: {}", health.is_ready().await?);
        }
        Err(e) => println!("Failed to get readiness status: {e}"),
    }

    // Get P2P information
    println!("\n=== P2P Information ===");
    let p2p = P2PClient::new(&client);
    match p2p.get_net_info().await {
        Ok(net_info) => {
            println!("Network ID: {}", net_info.id);
            println!("Listen addresses: {:?}", net_info.listen_addresses);
            println!("Connected peers: {:?}", net_info.connected_peers);
        }
        Err(e) => println!("Failed to get network info: {e}"),
    }

    match p2p.get_peer_info().await {
        Ok(peers) => {
            println!("Number of peers: {}", peers.len());
            for peer in peers {
                println!("  Peer ID: {}, Address: {}", peer.id, peer.address);
            }
        }
        Err(e) => println!("Failed to get peer info: {e}"),
    }

    // Get store information
    println!("\n=== Store Information ===");
    let store = StoreClient::new(&client);

    // Try to get the latest block (height 0 for genesis)
    match store.get_block_by_height(0).await {
        Ok(Some(block)) => {
            println!("Genesis block found:");
            println!(
                "  Height: {}",
                block
                    .header
                    .as_ref()
                    .map(|h| h.header.as_ref().map(|hdr| hdr.height).unwrap_or(0))
                    .unwrap_or(0)
            );
        }
        Ok(None) => println!("No block found at height 0"),
        Err(e) => println!("Failed to get block: {e}"),
    }

    // Get current state
    match store.get_state().await {
        Ok(Some(state)) => {
            println!("Current state found");
            println!("  Last block height: {}", state.last_block_height);
        }
        Ok(None) => println!("No state found"),
        Err(e) => println!("Failed to get state: {e}"),
    }

    // Get metadata by key
    match store.get_metadata("test_key".to_string()).await {
        Ok(value) => {
            println!("Metadata value: {}", hex::encode(&value));
        }
        Err(e) => println!("Failed to get metadata: {e}"),
    }

    Ok(())
}

// Helper function to encode bytes as hex (you'll need to add 'hex' to dev-dependencies)
mod hex {
    pub fn encode(bytes: &[u8]) -> String {
        bytes.iter().map(|b| format!("{b:02x}")).collect()
    }
}
