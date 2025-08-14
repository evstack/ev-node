pub mod compression;

pub mod v1 {
    // Always include the pure message types (no tonic dependencies)
    #[cfg(not(feature = "grpc"))]
    include!("proto/evnode.v1.messages.rs");

    // Include the full version with gRPC services when the feature is enabled
    #[cfg(feature = "grpc")]
    include!("proto/evnode.v1.services.rs");
}

// Re-export compression types for convenience
pub use compression::{
    compress_blob, decompress_blob, get_compression_info, BlobCompressor, CompressionError,
    CompressionInfo,
};
