# Build stage 1: Vite + Vue
FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Build stage 2: Go
FROM golang:1.26-alpine AS go-builder
WORKDIR /app
# Install dependencies
RUN apk add --no-cache gcc musl-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy frontend build to embed
COPY --from=frontend-builder /app/dist ./frontend/dist
# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o gostream main.go

# Runtime stage
FROM debian:bookworm-slim
# Install ffmpeg for normalisation, and ca-certificates for S3 HTTPS
RUN apt-get update && apt-get install -y ffmpeg ca-certificates nano procps && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=go-builder /app/gostream .

# Create a non-root user (optional but good practice)
RUN useradd -m -U gostream && mkdir -p /home/gostream/.config/gostream && chown -R gostream:gostream /home/gostream
USER gostream
ENV HOME=/home/gostream

EXPOSE 8080
CMD ["./gostream"]
