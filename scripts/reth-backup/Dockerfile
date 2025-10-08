FROM ghcr.io/evstack/ev-reth:latest

ARG LIBMDBX_REPO=https://github.com/erthink/libmdbx.git
ARG LIBMDBX_REF=master

RUN set -eux; \
    apt-get update; \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        cmake \
        git \
        jq \
    ; \
    rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    git clone --depth 1 --branch "${LIBMDBX_REF}" "${LIBMDBX_REPO}" /tmp/libmdbx; \
    cmake -S /tmp/libmdbx -B /tmp/libmdbx/build -DCMAKE_BUILD_TYPE=Release; \
    cmake --build /tmp/libmdbx/build --target mdbx_copy mdbx_dump mdbx_chk; \
    install -m 0755 /tmp/libmdbx/build/mdbx_copy /usr/local/bin/mdbx_copy; \
    install -m 0755 /tmp/libmdbx/build/mdbx_dump /usr/local/bin/mdbx_dump; \
    install -m 0755 /tmp/libmdbx/build/mdbx_chk /usr/local/bin/mdbx_chk; \
    rm -rf /tmp/libmdbx
