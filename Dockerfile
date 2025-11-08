# Dockerfile for Go MuPDF Wrapper Testing
# Uses latest Go on Debian/Ubuntu

FROM ubuntu:24.04

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive \
    GO_VERSION=1.24.2 \
    GOPATH=/go \
    PATH=/usr/local/go/bin:$PATH

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    gcc \
    g++ \
    make \
    pkg-config \
    git \
    curl \
    ca-certificates \
    libharfbuzz-dev \
    libfreetype6-dev \
    libjpeg-dev \
    libpng-dev \
    zlib1g-dev \
    libjbig2dec-dev \
    libopenjp2-7-dev \
    && rm -rf /var/lib/apt/lists/*

# Install latest Go
RUN curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz -o /tmp/go.tar.gz \
    && tar -C /usr/local -xzf /tmp/go.tar.gz \
    && rm /tmp/go.tar.gz \
    && go version

# Set working directory
WORKDIR /workspace

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Initialize git submodules (MuPDF)
RUN git submodule update --init --recursive || true

# Build MuPDF library from source
# This ensures we use the same MuPDF version regardless of Ubuntu package availability
RUN if [ -d "third_party/mupdf" ]; then \
        echo "Building MuPDF from source..." && \
        cd third_party/mupdf && \
        make -j$(nproc) libs && \
        cd ../.. && \
        echo "MuPDF build complete. Verifying libraries..." && \
        ls -la third_party/mupdf/build/release/*.a && \
        echo "Verifying headers..." && \
        ls -la third_party/mupdf/include/mupdf/*.h; \
    else \
        echo "Warning: third_party/mupdf directory not found. Submodule may not be initialized."; \
    fi

# Set up environment for CGO
# We use source-built MuPDF by default (no system_mupdf build tag)
ENV CGO_ENABLED=1 \
    PKG_CONFIG_PATH=/usr/lib/pkgconfig \
    MUPDF_BUILD_DIR=/workspace/third_party/mupdf

# Default command: run tests
CMD ["go", "test", "./pkg/mupdf/", "-v"]

