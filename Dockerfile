FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/sky-edge-balancer .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/sky-edge-balancer /sky-edge-balancer
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/sky-edge-balancer"]
