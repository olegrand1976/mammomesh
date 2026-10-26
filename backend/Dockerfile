# syntax=docker/dockerfile:1
# Binaire Go statique (API, orchestrateur, migrations) dans une image minimale sans shell.
FROM golang:1.26-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOFLAGS=-mod=readonly
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/mammomesh ./cmd/mammomesh

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/mammomesh /usr/local/bin/mammomesh
USER nonroot:nonroot
EXPOSE 8080 9090
ENTRYPOINT ["/usr/local/bin/mammomesh"]
CMD ["api"]
