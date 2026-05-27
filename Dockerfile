# Builder — native platform compiles for the target arch (no QEMU needed for Go)
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" \
    -o adguardhome-exporter ./cmd/adguardhome-exporter

# Final — scratch keeps the image minimal; CA certs are copied for HTTPS support
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/adguardhome-exporter /adguardhome-exporter

EXPOSE 9100

ENTRYPOINT ["/adguardhome-exporter"]
