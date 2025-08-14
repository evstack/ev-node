//! Blob compression and decompression module
//!
//! This module provides compression and decompression functionality for blobs,
//! matching the Go implementation in the ev-node DA layer.

use bytes::{Bytes, BytesMut};
use std::io;
use thiserror::Error;

/// Size of the compression header in bytes (1 byte flag + 8 bytes original size)
const COMPRESSION_HEADER_SIZE: usize = 9;

/// Compression flag for uncompressed data
const FLAG_UNCOMPRESSED: u8 = 0x00;

/// Compression flag for zstd compressed data  
const FLAG_ZSTD: u8 = 0x01;

/// Default zstd compression level
const DEFAULT_ZSTD_LEVEL: i32 = 3;

/// Compression-related errors
#[derive(Debug, Error)]
pub enum CompressionError {
    #[error("invalid compression header")]
    InvalidHeader,
    
    #[error("invalid compression flag: {0}")]
    InvalidCompressionFlag(u8),
    
    #[error("decompression failed: {0}")]
    DecompressionFailed(String),
    
    #[error("zstd error: {0}")]
    ZstdError(#[from] io::Error),
}

/// Result type for compression operations
pub type Result<T> = std::result::Result<T, CompressionError>;

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

/// Blob compressor/decompressor
pub struct BlobCompressor {
    /// Zstd compression level
    compression_level: i32,
}

impl BlobCompressor {
    /// Create a new blob compressor with default settings
    pub fn new() -> Self {
        Self {
            compression_level: DEFAULT_ZSTD_LEVEL,
        }
    }
    
    /// Create a new blob compressor with custom compression level
    pub fn with_level(compression_level: i32) -> Self {
        Self {
            compression_level,
        }
    }
    
    /// Compress a blob
    pub fn compress(&self, blob: &[u8]) -> Result<Bytes> {
        // For empty blobs, just add uncompressed header
        if blob.is_empty() {
            return Ok(self.add_compression_header(blob, FLAG_UNCOMPRESSED, 0));
        }
        
        // Try to compress with zstd
        let compressed = zstd::encode_all(blob, self.compression_level)?;
        
        // Check if compression is beneficial (at least 10% savings)
        let compression_ratio = compressed.len() as f64 / blob.len() as f64;
        if compression_ratio > 0.9 {
            // Compression not beneficial, store uncompressed
            Ok(self.add_compression_header(blob, FLAG_UNCOMPRESSED, blob.len() as u64))
        } else {
            // Compression beneficial
            Ok(self.add_compression_header(&compressed, FLAG_ZSTD, blob.len() as u64))
        }
    }
    
    /// Decompress a blob
    pub fn decompress(&self, compressed_blob: &[u8]) -> Result<Bytes> {
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
            
            let original_size = u64::from_le_bytes(
                compressed_blob[1..9].try_into().unwrap_or([0; 8])
            );
            
            // If flag is in printable ASCII range (32-126) and size is unreasonable,
            // it's likely a legacy text blob
            if (flag >= 32 && flag <= 126) && 
               (original_size == 0 || original_size > (compressed_blob.len() as u64 * 100)) {
                // Likely a legacy blob
                return Ok(Bytes::copy_from_slice(compressed_blob));
            }
            
            // Otherwise, it's likely a corrupted compressed blob
            return Err(CompressionError::InvalidCompressionFlag(flag));
        }
        
        // Parse the header
        let (flag, original_size, payload) = self.parse_compression_header(compressed_blob)?;
        
        match flag {
            FLAG_UNCOMPRESSED => {
                // Data is uncompressed, just return the payload
                Ok(Bytes::copy_from_slice(payload))
            }
            FLAG_ZSTD => {
                // Decompress with zstd
                let decompressed = zstd::decode_all(payload)
                    .map_err(|e| CompressionError::DecompressionFailed(e.to_string()))?;
                
                // Verify the decompressed size matches
                if decompressed.len() as u64 != original_size {
                    return Err(CompressionError::DecompressionFailed(
                        format!("size mismatch: expected {}, got {}", 
                               original_size, decompressed.len())
                    ));
                }
                
                Ok(Bytes::from(decompressed))
            }
            _ => {
                // Should not happen as we validated the flag earlier
                Err(CompressionError::InvalidCompressionFlag(flag))
            }
        }
    }
    
    /// Get compression information about a blob
    pub fn get_compression_info(&self, blob: &[u8]) -> CompressionInfo {
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
        
        if let Ok((flag, original_size, _)) = self.parse_compression_header(blob) {
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
    
    /// Add compression header to payload
    fn add_compression_header(&self, payload: &[u8], flag: u8, original_size: u64) -> Bytes {
        let mut result = BytesMut::with_capacity(COMPRESSION_HEADER_SIZE + payload.len());
        
        // Write flag
        result.extend_from_slice(&[flag]);
        
        // Write original size (little-endian)
        result.extend_from_slice(&original_size.to_le_bytes());
        
        // Write payload
        result.extend_from_slice(payload);
        
        result.freeze()
    }
    
    /// Parse compression header from blob
    fn parse_compression_header<'a>(&self, blob: &'a [u8]) -> Result<(u8, u64, &'a [u8])> {
        if blob.len() < COMPRESSION_HEADER_SIZE {
            return Err(CompressionError::InvalidHeader);
        }
        
        let flag = blob[0];
        let original_size = u64::from_le_bytes(
            blob[1..9].try_into().map_err(|_| CompressionError::InvalidHeader)?
        );
        let payload = &blob[COMPRESSION_HEADER_SIZE..];
        
        // Validate the compression flag
        if flag != FLAG_UNCOMPRESSED && flag != FLAG_ZSTD {
            return Err(CompressionError::InvalidCompressionFlag(flag));
        }
        
        Ok((flag, original_size, payload))
    }
}

impl Default for BlobCompressor {
    fn default() -> Self {
        Self::new()
    }
}

/// Convenience function to compress a blob with default settings
pub fn compress_blob(blob: &[u8]) -> Result<Bytes> {
    BlobCompressor::new().compress(blob)
}

/// Convenience function to decompress a blob
pub fn decompress_blob(compressed_blob: &[u8]) -> Result<Bytes> {
    BlobCompressor::new().decompress(compressed_blob)
}

/// Convenience function to get compression info about a blob
pub fn get_compression_info(blob: &[u8]) -> CompressionInfo {
    BlobCompressor::new().get_compression_info(blob)
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_compress_decompress_roundtrip() {
        let compressor = BlobCompressor::new();
        
        // Test with compressible data
        let original = b"hello world ".repeat(100);
        let compressed = compressor.compress(&original).unwrap();
        let decompressed = compressor.decompress(&compressed).unwrap();
        
        assert_eq!(original, decompressed.as_ref());
        
        // Verify it was actually compressed
        let info = compressor.get_compression_info(&compressed);
        assert!(info.is_compressed);
        assert_eq!(info.algorithm, "zstd");
        assert!(info.compression_ratio < 0.5); // Should compress well
    }
    
    #[test]
    fn test_uncompressed_fallback() {
        let compressor = BlobCompressor::new();
        
        // Random data that won't compress well
        let mut random_data = vec![0u8; 100];
        for i in 0..100 {
            random_data[i] = (i * 7 + 13) as u8; // Pseudo-random
        }
        
        let compressed = compressor.compress(&random_data).unwrap();
        let decompressed = compressor.decompress(&compressed).unwrap();
        
        assert_eq!(random_data, decompressed.as_ref());
        
        // Verify it was stored uncompressed
        let info = compressor.get_compression_info(&compressed);
        assert!(!info.is_compressed);
        assert_eq!(info.algorithm, "none");
    }
    
    #[test]
    fn test_legacy_blob() {
        let compressor = BlobCompressor::new();
        
        // Test with legacy blob (no compression header)
        let legacy_blob = b"legacy data without header";
        
        // Should return as-is
        let decompressed = compressor.decompress(legacy_blob).unwrap();
        assert_eq!(legacy_blob, decompressed.as_ref());
    }
    
    #[test]
    fn test_invalid_compression_flag() {
        let compressor = BlobCompressor::new();
        
        // Create blob with invalid flag
        let mut invalid_blob = vec![0u8; COMPRESSION_HEADER_SIZE + 10];
        invalid_blob[0] = 0xFF; // Invalid flag
        
        // Should return error
        let result = compressor.decompress(&invalid_blob);
        assert!(result.is_err());
        
        match result.unwrap_err() {
            CompressionError::InvalidCompressionFlag(flag) => {
                assert_eq!(flag, 0xFF);
            }
            _ => panic!("Expected InvalidCompressionFlag error"),
        }
    }
    
    #[test]
    fn test_empty_blob() {
        let compressor = BlobCompressor::new();
        
        let empty = vec![];
        let compressed = compressor.compress(&empty).unwrap();
        let decompressed = compressor.decompress(&compressed).unwrap();
        
        assert_eq!(empty, decompressed.as_ref());
    }
    
    #[test]
    fn test_compression_info() {
        let compressor = BlobCompressor::new();
        
        let original = b"compress me ".repeat(100);
        let compressed = compressor.compress(&original).unwrap();
        
        let info = compressor.get_compression_info(&compressed);
        assert!(info.is_compressed);
        assert_eq!(info.algorithm, "zstd");
        assert_eq!(info.original_size, original.len() as u64);
        assert!(info.compression_ratio < 1.0);
        assert!(info.compression_ratio > 0.0);
    }
}