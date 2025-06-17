FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh searcher

# Set working directory
WORKDIR /app

# Copy the binary
COPY k8s-slack-searcher /app/k8s-slack-searcher

# Create directories for data
RUN mkdir -p /app/source-data /app/databases && \
    chown -R searcher:searcher /app

# Switch to non-root user
USER searcher

# Expose volume for data
VOLUME ["/app/source-data", "/app/databases"]

ENTRYPOINT ["/app/k8s-slack-searcher"]