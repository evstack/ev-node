use crate::{client::Client, error::Result};

/// Health status of the node
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum HealthStatus {
    /// Node is operating normally
    Pass,
    /// Node is degraded but still serving
    Warn,
    /// Node has failed health checks
    Fail,
    /// Unknown health status
    Unknown,
}

pub struct HealthClient {
    base_url: String,
    http_client: reqwest::Client,
}

impl HealthClient {
    /// Create a new HealthClient from a Client
    ///
    /// Note: The base_url should be the HTTP endpoint (e.g., "http://localhost:9090")
    pub fn new(_client: &Client) -> Self {
        // For now, we'll need to construct the base URL from the client
        // This is a workaround since we're mixing gRPC and HTTP endpoints
        // TODO: Consider adding a method to Client to get the base URL
        Self::with_base_url("http://localhost:9090".to_string())
    }

    /// Create a new HealthClient with an explicit base URL
    pub fn with_base_url(base_url: String) -> Self {
        Self {
            base_url: base_url.trim_end_matches('/').to_string(),
            http_client: reqwest::Client::new(),
        }
    }

    /// Check if the node is alive and get its health status
    pub async fn livez(&self) -> Result<HealthStatus> {
        let url = format!("{}/health/live", self.base_url);
        let response = self.http_client.get(&url).send().await?;

        let status_text = response.text().await?.trim().to_string();

        match status_text.as_str() {
            "OK" => Ok(HealthStatus::Pass),
            "WARN" => Ok(HealthStatus::Warn),
            "FAIL" => Ok(HealthStatus::Fail),
            _ => Ok(HealthStatus::Unknown),
        }
    }

    /// Check if the node is healthy (status is PASS)
    pub async fn is_healthy(&self) -> Result<bool> {
        let status = self.livez().await?;
        Ok(status == HealthStatus::Pass)
    }
}
