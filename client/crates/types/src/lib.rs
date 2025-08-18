#![cfg_attr(not(feature = "std"), no_std)]

// When no_std, we need alloc for prost's Vec and String types
#[cfg(not(feature = "std"))]
extern crate alloc;

#[cfg(feature = "compression")]
pub mod compression;

pub mod v1 {
    // Include the generated protobuf code
    // The generated code has feature gates for client and server code
    include!("proto/evnode.v1.rs");
}

// Re-export compression types for convenience when compression is enabled
#[cfg(feature = "compression")]
pub use compression::{
    decompress_blob, get_compression_info, CompressionError, CompressionInfo,
};
