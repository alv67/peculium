FROM golang:1.27-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.24
RUN apk add --no-cache ca-certificates tzdata postgresql16-client
COPY --from=builder /server /server
COPY --from=builder /app/migrations /migrations

ARG PECULIUM_VERSION=dev
ARG PECULIUM_COMMIT=""
ARG PECULIUM_BUILT_AT=""
ENV PECULIUM_VERSION="${PECULIUM_VERSION}"
ENV PECULIUM_COMMIT="${PECULIUM_COMMIT}"
ENV PECULIUM_BUILT_AT="${PECULIUM_BUILT_AT}"

WORKDIR /
EXPOSE 8080
