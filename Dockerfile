FROM node:18-alpine AS frontend-builder

WORKDIR /workspace

# Copy web directory
COPY web ./web

# Install dependencies and build
RUN cd web && npm ci && npm run build

FROM golang:1.24-alpine AS builder

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend from frontend-builder (vite outputs to ../internal/api/static from web/)
COPY --from=frontend-builder /workspace/internal/api/static ./internal/api/static

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -o manager cmd/main.go

# Use distroless as minimal base image
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
