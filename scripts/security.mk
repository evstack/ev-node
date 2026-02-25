# security.mk - Security scanning with Trivy (https://trivy.dev)

TRIVY_IMAGE := aquasec/trivy:latest
TRIVY_SEVERITY ?= CRITICAL,HIGH
TRIVY_CACHE_VOLUME := trivy-cache

# Docker images to scan (space-separated, override or extend as needed)
SCAN_IMAGES ?= evstack:local-dev

# Common docker run args for Trivy
TRIVY_RUN := docker run --rm \
	-v $(TRIVY_CACHE_VOLUME):/root/.cache/ \
	-e TRIVY_SEVERITY=$(TRIVY_SEVERITY)

## trivy-scan: Run all Trivy security scans (filesystem + Docker images)
trivy-scan: trivy-scan-fs trivy-scan-image
.PHONY: trivy-scan

## trivy-scan-fs: Scan repo for dependency vulnerabilities, misconfigs, and secrets
trivy-scan-fs:
	@echo "--> Scanning repository filesystem with Trivy"
	@$(TRIVY_RUN) \
		-v $(CURDIR):/workspace \
		$(TRIVY_IMAGE) \
		fs --scanners vuln,misconfig,secret \
		--severity $(TRIVY_SEVERITY) \
		/workspace
	@echo "--> Filesystem scan complete"
.PHONY: trivy-scan-fs

## trivy-scan-image: Scan built Docker images for vulnerabilities
trivy-scan-image:
	@echo "--> Scanning Docker images with Trivy"
	@for img in $(SCAN_IMAGES); do \
		echo "--> Scanning image: $$img"; \
		$(TRIVY_RUN) \
			-v /var/run/docker.sock:/var/run/docker.sock \
			$(TRIVY_IMAGE) \
			image --severity $(TRIVY_SEVERITY) \
			$$img; \
	done
	@echo "--> Image scan complete"
.PHONY: trivy-scan-image
