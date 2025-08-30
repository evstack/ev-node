//! Evolve Rust Client Library
//!
//! This library provides a Rust client for interacting with Evolve nodes via gRPC.
//!
//! # Example
//!
//! ```no_run
//! use ev_client::{Client, HealthClient, ConfigClient};
//!
//! #[tokio::main]
//! async fn main() -> Result<(), Box<dyn std::error::Error>> {
//!     // Connect to a Evolve node
//!     let client = Client::connect("http://localhost:50051").await?;
//!     
//!     // Check health
//!     let health = HealthClient::new(&client);
//!     let is_healthy = health.is_healthy().await?;
//!     println!("Node healthy: {}", is_healthy);
//!     
//!     // Get namespace configuration
//!     let config = ConfigClient::new(&client);
//!     let namespace = config.get_namespace().await?;
//!     println!("Header namespace: {}", namespace.header_namespace);
//!     println!("Data namespace: {}", namespace.data_namespace);
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
//!         .endpoint("https://secure-node.ev.xyz")
//!         .tls()  // Enable TLS with default configuration
//!         .build()
//!         .await?;
//!     
//!     // Or with custom TLS configuration
//!     let tls_config = ClientTlsConfig::new()
//!         .domain_name("secure-node.ev.xyz");
//!     
//!     let client = Client::builder()
//!         .endpoint("https://secure-node.ev.xyz")
//!         .tls_config(tls_config)
//!         .build()
//!         .await?;
//!     
//!     Ok(())
//! }
//! ```

pub mod client;
pub mod config;
pub mod error;
pub mod health;
pub mod p2p;
pub mod signer;
pub mod store;

// Re-export main types for convenience
pub use client::{Client, ClientBuilder};
pub use config::ConfigClient;
pub use error::{ClientError, Result};
pub use health::HealthClient;
pub use p2p::P2PClient;
pub use signer::SignerClient;
pub use store::StoreClient;

// Re-export types from evolve-types for convenience
pub use ev_types::v1;

// Re-export tonic transport types for convenience
pub use tonic::transport::ClientTlsConfig;
