//! Blob decompression module
//!
//! This module provides decompression functionality for blobs,
//! matching the Go implementation in the ev-node DA layer.
//! Uses ruzstd for pure Rust zstd decompression without C dependencies.

use bytes::Bytes;
use ruzstd::decoding::StreamingDecoder;
use snafu::Snafu;

#[cfg(not(feature = "std"))]
use alloc::{format, string::{String, ToString}, vec::Vec};

#[cfg(not(feature = "std"))]
use ruzstd::io::Read;

#[cfg(feature = "std")]
use std::io::Read;

/// Size of the compression header in bytes (1 byte flag + 8 bytes original size)
const COMPRESSION_HEADER_SIZE: usize = 9;

/// Compression flag for uncompressed data
const FLAG_UNCOMPRESSED: u8 = 0x00;

/// Compression flag for zstd compressed data
const FLAG_ZSTD: u8 = 0x01;

/// Compression-related errors
#[derive(Debug, Snafu)]
pub enum CompressionError {
    #[snafu(display("invalid compression header"))]
    InvalidHeader,

    #[snafu(display("invalid compression flag: {flag}"))]
    InvalidCompressionFlag { flag: u8 },

    #[snafu(display("decompression failed: {message}"))]
    DecompressionFailed { message: String },
}

/// Result type for compression operations
pub type Result<T> = core::result::Result<T, CompressionError>;

/// Information about a compressed blob
#[derive(Debug, Clone)]
pub struct CompressionInfo {
    /// Whether the blob is compressed
    pub is_compressed: bool,
    /// Compression algorithm used ("none", "zstd")
    pub algorithm: String,
    /// Original size before compression
    pub original_size: u64,
    /// Compressed size
    pub compressed_size: usize,
    /// Compression ratio (compressed_size / original_size)
    pub compression_ratio: f64,
}

/// Parse compression header from blob
fn parse_compression_header(blob: &[u8]) -> Result<(u8, u64, &[u8])> {
    if blob.len() < COMPRESSION_HEADER_SIZE {
        return Err(CompressionError::InvalidHeader);
    }

    let flag = blob[0];
    let original_size = u64::from_le_bytes(
        blob[1..9]
            .try_into()
            .map_err(|_| CompressionError::InvalidHeader)?,
    );
    let payload = &blob[COMPRESSION_HEADER_SIZE..];

    // Validate the compression flag
    if flag != FLAG_UNCOMPRESSED && flag != FLAG_ZSTD {
        return Err(CompressionError::InvalidCompressionFlag { flag });
    }

    Ok((flag, original_size, payload))
}

/// Decompress a blob
pub fn decompress_blob(compressed_blob: &[u8]) -> Result<Bytes> {
    // Check if blob is too small to have a header
    if compressed_blob.len() < COMPRESSION_HEADER_SIZE {
        // Assume legacy uncompressed blob
        return Ok(Bytes::copy_from_slice(compressed_blob));
    }

    // Check the compression flag
    let flag = compressed_blob[0];

    // Handle invalid flags with legacy blob heuristics
    if flag != FLAG_UNCOMPRESSED && flag != FLAG_ZSTD {
        // This could be either a legacy blob or a corrupted header
        // Use heuristics to determine which

        let original_size = u64::from_le_bytes(compressed_blob[1..9].try_into().unwrap_or([0; 8]));

        // If flag is in printable ASCII range (32-126) and size is unreasonable,
        // it's likely a legacy text blob
        if (32..=126).contains(&flag)
            && (original_size == 0 || original_size > (compressed_blob.len() as u64 * 100))
        {
            // Likely a legacy blob
            return Ok(Bytes::copy_from_slice(compressed_blob));
        }

        // Otherwise, it's likely a corrupted compressed blob
        return Err(CompressionError::InvalidCompressionFlag { flag });
    }

    // Parse the header
    let (flag, original_size, payload) = parse_compression_header(compressed_blob)?;

    match flag {
        FLAG_UNCOMPRESSED => {
            // Data is uncompressed, just return the payload
            Ok(Bytes::copy_from_slice(payload))
        }
        FLAG_ZSTD => {
            // Decompress with ruzstd
            let mut decoder = StreamingDecoder::new(payload).map_err(|e| {
                CompressionError::DecompressionFailed {
                    message: format!("{}", e),
                }
            })?;

            let mut decompressed = Vec::new();
            decoder.read_to_end(&mut decompressed).map_err(|e| {
                CompressionError::DecompressionFailed {
                    message: format!("{}", e),
                }
            })?;

            // Verify the decompressed size matches
            if decompressed.len() as u64 != original_size {
                return Err(CompressionError::DecompressionFailed {
                    message: format!(
                        "size mismatch: expected {}, got {}",
                        original_size,
                        decompressed.len()
                    ),
                });
            }

            Ok(Bytes::from(decompressed))
        }
        _ => {
            // Should not happen as we validated the flag earlier
            Err(CompressionError::InvalidCompressionFlag { flag })
        }
    }
}

/// Get compression information about a blob
pub fn get_compression_info(blob: &[u8]) -> CompressionInfo {
    if blob.len() < COMPRESSION_HEADER_SIZE {
        return CompressionInfo {
            is_compressed: false,
            algorithm: "none".to_string(),
            original_size: blob.len() as u64,
            compressed_size: blob.len(),
            compression_ratio: 1.0,
        };
    }

    let flag = blob[0];
    if flag != FLAG_UNCOMPRESSED && flag != FLAG_ZSTD {
        // Legacy or invalid blob
        return CompressionInfo {
            is_compressed: false,
            algorithm: "none".to_string(),
            original_size: blob.len() as u64,
            compressed_size: blob.len(),
            compression_ratio: 1.0,
        };
    }

    if let Ok((flag, original_size, _)) = parse_compression_header(blob) {
        let algorithm = match flag {
            FLAG_UNCOMPRESSED => "none",
            FLAG_ZSTD => "zstd",
            _ => "unknown",
        };

        CompressionInfo {
            is_compressed: flag == FLAG_ZSTD,
            algorithm: algorithm.to_string(),
            original_size,
            compressed_size: blob.len(),
            compression_ratio: if original_size > 0 {
                blob.len() as f64 / original_size as f64
            } else {
                1.0
            },
        }
    } else {
        CompressionInfo {
            is_compressed: false,
            algorithm: "none".to_string(),
            original_size: blob.len() as u64,
            compressed_size: blob.len(),
            compression_ratio: 1.0,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_legacy_blob() {
        // Test with legacy blob (no compression header)
        let legacy_blob = b"legacy data without header";

        // Should return as-is
        let decompressed = decompress_blob(legacy_blob).unwrap();
        assert_eq!(legacy_blob, decompressed.as_ref());
    }

    #[test]
    fn test_invalid_compression_flag() {
        // Create blob with invalid flag
        let mut invalid_blob = vec![0u8; COMPRESSION_HEADER_SIZE + 10];
        invalid_blob[0] = 0xFF; // Invalid flag

        // Should return error
        let result = decompress_blob(&invalid_blob);
        assert!(result.is_err());

        match result.unwrap_err() {
            CompressionError::InvalidCompressionFlag { flag } => {
                assert_eq!(flag, 0xFF);
            }
            _ => panic!("Expected InvalidCompressionFlag error"),
        }
    }

    #[test]
    fn test_uncompressed_with_header() {
        // Create a blob with uncompressed header
        let original_data = b"test data";
        let mut blob = Vec::with_capacity(COMPRESSION_HEADER_SIZE + original_data.len());

        // Add header
        blob.push(FLAG_UNCOMPRESSED);
        blob.extend_from_slice(&(original_data.len() as u64).to_le_bytes());
        blob.extend_from_slice(original_data);

        // Decompress
        let decompressed = decompress_blob(&blob).unwrap();
        assert_eq!(original_data, decompressed.as_ref());

        // Check info
        let info = get_compression_info(&blob);
        assert!(!info.is_compressed);
        assert_eq!(info.algorithm, "none");
        assert_eq!(info.original_size, original_data.len() as u64);
    }

    #[test]
    fn test_compression_info() {
        // Test with uncompressed data
        let mut blob = Vec::new();
        blob.push(FLAG_UNCOMPRESSED);
        blob.extend_from_slice(&100u64.to_le_bytes());
        blob.extend_from_slice(&[0u8; 100]);

        let info = get_compression_info(&blob);
        assert!(!info.is_compressed);
        assert_eq!(info.algorithm, "none");
        assert_eq!(info.original_size, 100);
    }
}
