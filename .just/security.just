# Security scanning with Trivy (https://trivy.dev)

trivy_image := "aquasec/trivy:latest"
trivy_severity := env("TRIVY_SEVERITY", "CRITICAL,HIGH")
trivy_cache_volume := "trivy-cache"
scan_images := env("SCAN_IMAGES", "evstack:local-dev")

trivy_run := "docker run --rm -v " + trivy_cache_volume + ":/root/.cache/ -e TRIVY_SEVERITY=" + trivy_severity

# Run all Trivy security scans (filesystem + Docker images)
trivy-scan: trivy-scan-fs trivy-scan-image

# Scan repo for dependency vulnerabilities, misconfigs, and secrets
trivy-scan-fs:
    @echo "--> Scanning repository filesystem with Trivy"
    @{{trivy_run}} \
        -v {{justfile_directory()}}:/workspace \
        {{trivy_image}} \
        fs --scanners vuln,misconfig,secret \
        --severity {{trivy_severity}} \
        /workspace
    @echo "--> Filesystem scan complete"

# Scan built Docker images for vulnerabilities
trivy-scan-image:
    @echo "--> Scanning Docker images with Trivy"
    @for img in {{scan_images}}; do \
        echo "--> Scanning image: $img"; \
        {{trivy_run}} \
            -v /var/run/docker.sock:/var/run/docker.sock \
            {{trivy_image}} \
            image --severity {{trivy_severity}} \
            $img; \
    done
    @echo "--> Image scan complete"
