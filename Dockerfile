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
# Install ffmpeg for normalisation, ca-certificates for S3 HTTPS, and gosu for entrypoint user drop
RUN apt-get update && apt-get install -y ffmpeg ca-certificates gosu && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=go-builder /app/gostream .
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh \
	&& useradd -m -U gostream \
	&& mkdir -p /home/gostream/.config/gostream /home/gostream/.local/share/gostream \
	&& chown -R gostream:gostream /home/gostream /app/gostream

ENV HOME=/home/gostream

EXPOSE 8080
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["./gostream"]
