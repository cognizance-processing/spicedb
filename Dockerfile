FROM golang:1.22.4-alpine3.19 AS spicedb-builder
WORKDIR /go/src/app
RUN apk update && apk add --no-cache git
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg/mod CGO_ENABLED=0 go build -v ./cmd/...

FROM golang:1.22.4-alpine3.19 AS health-probe-builder
WORKDIR /go/src/app
RUN apk update && apk add --no-cache git
RUN git clone https://github.com/grpc-ecosystem/grpc-health-probe.git
WORKDIR /go/src/app/grpc-health-probe
RUN git checkout bea3bb2419f2d0f0cd4a97b8190e8fafb3e48dda
RUN CGO_ENABLED=0 go install -a -tags netgo -ldflags=-w

FROM cgr.dev/chainguard/static:latest
#COPY --from=ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.20 /ko-app/grpc-health-probe /usr/local/bin/grpc_health_probe
#COPY --from=health-probe-builder /go/bin/grpc-health-probe /bin/grpc_health_probe
#COPY --from=spicedb-builder /go/src/app/spicedb /usr/local/bin/spicedb
ENV PATH="$PATH:/usr/local/bin"
EXPOSE 50051
ENTRYPOINT ["/app/spicedb", "serve", "--grpc-preshared-key", "b2601263774ff8e988057acfac2b6d769297dfdf19206fbefbf60a0b02e10569","--dashboard-enabled=false", "--datastore-engine=postgres", "--datastore-conn-uri=\"postgres://cloudRunUser:<~4%J6}8p7F:F{nS@34.44.110.10:5432/spicedb?sslmode=disable\""]
