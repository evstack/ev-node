//! Comprehensive tests for blob decompression
//! These tests match the behavior of the Go implementation

use ev_types::{decompress_blob, get_compression_info, CompressionError};

#[test]
fn test_legacy_blobs() {
    // Test with legacy blob (no compression header)
    let legacy_blob = b"legacy data without header";

    // Should return as-is
    let decompressed = decompress_blob(legacy_blob).unwrap();
    assert_eq!(legacy_blob, decompressed.as_ref());
}

#[test]
fn test_invalid_compression_flag() {
    // Create blob with invalid flag
    let mut invalid_blob = vec![0u8; 9 + 10]; // COMPRESSION_HEADER_SIZE + 10
    invalid_blob[0] = 0xFF; // Invalid flag

    // Should return error
    let result = decompress_blob(&invalid_blob);
    assert!(result.is_err());

    if let Err(CompressionError::InvalidCompressionFlag { flag }) = result {
        assert_eq!(flag, 0xFF);
    } else {
        panic!("Expected InvalidCompressionFlag error");
    }
}

#[test]
fn test_uncompressed_with_header() {
    // Create a blob with uncompressed header
    let original_data = b"test data";
    let mut blob = Vec::with_capacity(9 + original_data.len());

    // Add header (flag + 8 bytes for size)
    blob.push(0x00); // FLAG_UNCOMPRESSED
    blob.extend_from_slice(&(original_data.len() as u64).to_le_bytes());
    blob.extend_from_slice(original_data);

    // Decompress
    let decompressed = decompress_blob(&blob).unwrap();
    assert_eq!(original_data, decompressed.as_ref());
}

#[test]
fn test_compression_info() {
    // Test with uncompressed data
    let original_data = b"test data";
    let mut blob = Vec::new();
    blob.push(0x00); // FLAG_UNCOMPRESSED
    blob.extend_from_slice(&(original_data.len() as u64).to_le_bytes());
    blob.extend_from_slice(original_data);

    let info = get_compression_info(&blob);
    assert!(!info.is_compressed);
    assert_eq!(info.algorithm, "none");
    assert_eq!(info.original_size, original_data.len() as u64);
}

#[test]
fn test_empty_blob() {
    let empty = vec![];

    // Should handle empty blob gracefully
    let decompressed = decompress_blob(&empty).unwrap();
    assert_eq!(empty, decompressed.as_ref());
}

#[test]
fn test_legacy_blob_heuristics() {
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

        let decompressed = decompress_blob(&padded_blob).unwrap();
        assert_eq!(padded_blob, decompressed.as_ref());
    }
}

#[test]
fn test_corrupted_blob_detection() {
    // Create a blob that looks like it has a header but is corrupted
    let mut corrupted = vec![0u8; 20];
    corrupted[0] = 0xAB; // Invalid flag that's not ASCII
                         // Set a reasonable size that suggests this was meant to be compressed
    let size_bytes = 1000u64.to_le_bytes();
    corrupted[1..9].copy_from_slice(&size_bytes);

    // Should detect as corrupted
    let result = decompress_blob(&corrupted);
    assert!(result.is_err());
}
