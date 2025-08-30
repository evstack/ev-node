use crate::{client::Client, error::Result};
use ev_types::v1::{
    config_service_client::ConfigServiceClient, GetNamespaceResponse, GetSignerInfoResponse,
};
use tonic::Request;

pub struct ConfigClient {
    inner: ConfigServiceClient<tonic::transport::Channel>,
}

impl ConfigClient {
    /// Create a new ConfigClient from a Client
    pub fn new(client: &Client) -> Self {
        let inner = ConfigServiceClient::new(client.channel().clone());
        Self { inner }
    }

    /// Get the namespace for this network
    pub async fn get_namespace(&self) -> Result<GetNamespaceResponse> {
        let request = Request::new(());
        let response = self.inner.clone().get_namespace(request).await?;

        Ok(response.into_inner())
    }

    /// Get SequencerInfo
    pub async fn get_signer_info(&self) -> Result<GetSignerInfoResponse> {
        let request = Request::new(());
        let response = self.inner.clone().get_signer_info(request).await?;

        Ok(response.into_inner())
    }

    /// Get the header namespace
    pub async fn get_header_namespace(&self) -> Result<String> {
        let response = self.get_namespace().await?;
        Ok(response.header_namespace)
    }

    /// Get the data namespace
    pub async fn get_data_namespace(&self) -> Result<String> {
        let response = self.get_namespace().await?;
        Ok(response.data_namespace)
    }
}
