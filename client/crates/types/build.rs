use std::{env, fs, path::Path};
use walkdir::WalkDir;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let manifest_dir_str = env::var("CARGO_MANIFEST_DIR")?;
    let manifest_dir = Path::new(&manifest_dir_str);

    // Create output directories
    let proto_dir = manifest_dir.join("src/proto");
    fs::create_dir_all(&proto_dir)?;

    // Check if generated file already exists
    let generated_file = proto_dir.join("evnode.v1.rs");

    // Check for environment variable to force regeneration
    let force_regen = env::var("EV_TYPES_FORCE_PROTO_GEN").is_ok();

    // If file exists and we're not forcing regeneration, skip generation
    if !force_regen && generated_file.exists() {
        println!("cargo:warning=Using pre-generated proto files. Set EV_TYPES_FORCE_PROTO_GEN=1 to regenerate.");
        return Ok(());
    }

    // Make the include dir absolute and resolved (no "..", symlinks, etc.)
    let proto_root = match manifest_dir.join("../../../proto").canonicalize() {
        Ok(path) => path,
        Err(e) => {
            // If proto files don't exist but generated file does, that's ok
            if generated_file.exists() {
                println!("cargo:warning=Proto source files not found at ../../../proto, using pre-generated files");
                return Ok(());
            }
            // Otherwise, this is a real error
            return Err(
                format!("Proto files not found and no pre-generated files available: {e}").into(),
            );
        }
    };

    // Collect the .proto files
    let proto_files: Vec<_> = WalkDir::new(&proto_root)
        .into_iter()
        .filter_map(|e| {
            let p = e.ok()?.into_path();
            (p.extension()?.to_str()? == "proto").then_some(p)
        })
        .collect();

    // Generate a single file with proper feature gates for server and client code
    tonic_prost_build::configure()
        .build_server(true)
        .build_client(true)
        // Add cfg attributes to gate both server and client code behind the "grpc" feature
        .server_mod_attribute(".", "#[cfg(feature = \"grpc\")]")
        .client_mod_attribute(".", "#[cfg(feature = \"grpc\")]")
        // Use BTreeMap instead of HashMap for no_std compatibility
        .btree_map(".")
        // Generate to our output directory
        .out_dir(&proto_dir)
        .compile_protos(&proto_files, std::slice::from_ref(&proto_root.clone()))?;

    println!("cargo:rerun-if-changed={}", proto_root.display());
    Ok(())
}
