//! ev-node Rust Client Library
//!
//! This library provides a Rust client for interacting with ev-node nodes via gRPC.
//!
//! # Example
//!
//! ```no_run
//! use ev_client::{Client, HealthClient};
//!
//! #[tokio::main]
//! async fn main() -> Result<(), Box<dyn std::error::Error>> {
//!     // Connect to an ev-node node
//!     let client = Client::connect("http://localhost:50051").await?;
//!     
//!     // Check health
//!     let health = HealthClient::new(&client);
//!     let is_healthy = health.is_healthy().await?;
//!     println!("Node healthy: {}", is_healthy);
//!     
//!     Ok(())
//! }
//! ```
//!
//! # Using the Builder Pattern
//!
//! ```no_run
//! use ev_client::Client;
//! use std::time::Duration;
//!
//! #[tokio::main]
//! async fn main() -> Result<(), Box<dyn std::error::Error>> {
//!     // Create a client with custom timeouts
//!     let client = Client::builder()
//!         .endpoint("http://localhost:50051")
//!         .timeout(Duration::from_secs(30))
//!         .connect_timeout(Duration::from_secs(10))
//!         .build()
//!         .await?;
//!     
//!     Ok(())
//! }
//! ```
//!
//! # Using TLS
//!
//! ```no_run
//! use ev_client::Client;
//! use tonic::transport::ClientTlsConfig;
//!
//! #[tokio::main]
//! async fn main() -> Result<(), Box<dyn std::error::Error>> {
//!     // Create a client with TLS enabled
//!     let client = Client::builder()
//!         .endpoint("https://secure-node.evnode.dev")
//!         .tls()  // Enable TLS with default configuration
//!         .build()
//!         .await?;
//!     
//!     // Or with custom TLS configuration
//!     let tls_config = ClientTlsConfig::new()
//!         .domain_name("secure-node.evnode.dev");
//!     
//!     let client = Client::builder()
//!         .endpoint("https://secure-node.evnode.dev")
//!         .tls_config(tls_config)
//!         .build()
//!         .await?;
//!     
//!     Ok(())
//! }
//! ```

pub mod client;
pub mod error;
pub mod health;
pub mod p2p;
pub mod signer;
pub mod store;

// Re-export main types for convenience
pub use client::{Client, ClientBuilder};
pub use error::{ClientError, Result};
pub use health::HealthClient;
pub use p2p::P2PClient;
pub use signer::SignerClient;
pub use store::StoreClient;

// Re-export types from ev-types for convenience
pub use ev_types::v1;

// Re-export tonic transport types for convenience
pub use tonic::transport::ClientTlsConfig;
