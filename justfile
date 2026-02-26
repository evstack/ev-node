# ev-node justfile

# Extract version information from git
version := `git describe --tags --abbrev=0 2>/dev/null || echo "dev"`
gitsha := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
ldflags := "-X github.com/evstack/ev-node/pkg/cmd.Version=" + version + " -X github.com/evstack/ev-node/pkg/cmd.GitSHA=" + gitsha

# Tool build ldflags (uses main package)
tool_ldflags := "-X main.Version=" + version + " -X main.GitSHA=" + gitsha

# Build directory
build_dir := justfile_directory() / "build"

import '.just/help.just'
import '.just/build.just'
import '.just/dev.just'
import '.just/gen.just'
