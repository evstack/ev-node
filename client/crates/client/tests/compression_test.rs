//! Comprehensive tests for blob compression/decompression
//! These tests match the behavior of the Go implementation

use ev_client::compression::*;
use ev_client::{compress_blob, decompress_blob, get_compression_info};

#[test]
fn test_zstd_compression() {
    let compressor = BlobCompressor::with_level(3);
    
    // Test with compressible data
    let original_data: Vec<u8> = "hello world ".repeat(100).into_bytes();
    
    let compressed = compressor.compress(&original_data).unwrap();
    
    // Check that compression header is present
    assert!(compressed.len() >= 9); // COMPRESSION_HEADER_SIZE
    
    // Verify compression flag
    assert_eq!(compressed[0], 0x01); // FLAG_ZSTD
    
    // Decompress and verify
    let decompressed = compressor.decompress(&compressed).unwrap();
    assert_eq!(original_data, decompressed.as_ref());
}

#[test]
fn test_uncompressed_fallback() {
    let compressor = BlobCompressor::with_level(3);
    
    // Generate pseudo-random data that won't compress well
    let mut random_data = Vec::with_capacity(100);
    for i in 0..100 {
        random_data.push(((i * 17 + 23) % 256) as u8);
    }
    
    let compressed = compressor.compress(&random_data).unwrap();
    
    // Should use uncompressed flag
    assert_eq!(compressed[0], 0x00); // FLAG_UNCOMPRESSED
    
    // Decompress and verify
    let decompressed = compressor.decompress(&compressed).unwrap();
    assert_eq!(random_data, decompressed.as_ref());
}

#[test]
fn test_legacy_blobs() {
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
    let mut invalid_blob = vec![0u8; 9 + 10]; // COMPRESSION_HEADER_SIZE + 10
    invalid_blob[0] = 0xFF; // Invalid flag
    
    // Should return error
    let result = compressor.decompress(&invalid_blob);
    assert!(result.is_err());
    
    if let Err(CompressionError::InvalidCompressionFlag(flag)) = result {
        assert_eq!(flag, 0xFF);
    } else {
        panic!("Expected InvalidCompressionFlag error");
    }
}

#[test]
fn test_compression_info() {
    let compressor = BlobCompressor::new();
    
    // Test with compressible data
    let original_data: Vec<u8> = "compress me ".repeat(100).into_bytes();
    
    let compressed = compressor.compress(&original_data).unwrap();
    
    let info = compressor.get_compression_info(&compressed);
    assert!(info.is_compressed);
    assert_eq!(info.algorithm, "zstd");
    assert_eq!(info.original_size, original_data.len() as u64);
    assert!(info.compression_ratio < 1.0);
    assert!(info.compression_ratio > 0.0);
}

#[test]
fn test_helper_functions() {
    let original_data: Vec<u8> = "test ".repeat(100).into_bytes();
    
    // Test standalone compress function
    let compressed = compress_blob(&original_data).unwrap();
    
    // Test standalone decompress function
    let decompressed = decompress_blob(&compressed).unwrap();
    
    assert_eq!(original_data, decompressed.as_ref());
    
    // Test info function
    let info = get_compression_info(&compressed);
    assert!(info.is_compressed);
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
fn test_large_blob_efficiency() {
    let compressor = BlobCompressor::new();
    
    // Test different data types
    let test_cases = vec![
        ("repetitive", vec![b'A'; 100_000]),
        ("json", r#"{"key": "value", "array": [1, 2, 3], "nested": {"foo": "bar"}}"#.repeat(1000).into_bytes()),
        ("text", "The quick brown fox jumps over the lazy dog. ".repeat(1000).into_bytes()),
    ];
    
    for (name, data) in test_cases {
        let compressed = compressor.compress(&data).unwrap();
        let info = compressor.get_compression_info(&compressed);
        
        println!("{}: Original={}, Compressed={}, Ratio={:.3}, Algorithm={}", 
                 name, data.len(), info.compressed_size, info.compression_ratio, info.algorithm);
        
        // Verify round-trip
        let decompressed = compressor.decompress(&compressed).unwrap();
        assert_eq!(data, decompressed.as_ref(), "Failed for {}", name);
        
        // Repetitive and text data should compress well
        if name != "random" {
            assert!(info.compression_ratio < 0.8, "{} should compress well", name);
        }
    }
}

#[test]
fn test_legacy_blob_heuristics() {
    let compressor = BlobCompressor::new();
    
    // Test various legacy blobs that should be detected
    let legacy_blobs = vec![
        b"plain text data".to_vec(),
        b"JSON: {\"key\": \"value\"}".to_vec(),
        b"log entry: 2024-01-01 00:00:00 INFO message".to_vec(),
    ];
    
    for blob in legacy_blobs {
        // Ensure blob is large enough to potentially have a header
        let mut padded_blob = blob.clone();
        while padded_blob.len() < 20 {
            padded_blob.push(b' ');
        }
        
        let decompressed = compressor.decompress(&padded_blob).unwrap();
        assert_eq!(padded_blob, decompressed.as_ref());
    }
}

#[test] 
fn test_corrupted_blob_detection() {
    let compressor = BlobCompressor::new();
    
    // Create a blob that looks like it has a header but is corrupted
    let mut corrupted = vec![0u8; 20];
    corrupted[0] = 0xAB; // Invalid flag that's not ASCII
    // Set a reasonable size that suggests this was meant to be compressed
    let size_bytes = 1000u64.to_le_bytes();
    corrupted[1..9].copy_from_slice(&size_bytes);
    
    // Should detect as corrupted
    let result = compressor.decompress(&corrupted);
    assert!(result.is_err());
}

#[test]
fn test_compression_with_different_levels() {
    let data: Vec<u8> = "compress this data please ".repeat(100).into_bytes();
    
    // Test different compression levels
    let levels = vec![1, 3, 5, 9];
    let mut sizes = vec![];
    
    for level in levels {
        let compressor = BlobCompressor::with_level(level);
        let compressed = compressor.compress(&data).unwrap();
        sizes.push((level, compressed.len()));
        
        // Verify decompression works
        let decompressed = compressor.decompress(&compressed).unwrap();
        assert_eq!(data, decompressed.as_ref());
    }
    
    // Higher compression levels should generally produce smaller output
    println!("Compression level comparison:");
    for (level, size) in sizes {
        println!("  Level {}: {} bytes", level, size);
    }
}