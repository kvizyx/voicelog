FROM golang:1.22-alpine as builder

RUN apk update && apk add --no-cache ca-certificates && update-ca-certificates

ENV USER=appuser
ENV UID=10001

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download
RUN go mod verify

COPY cmd cmd
COPY internal internal
COPY pkg pkg
COPY .env .env
COPY .tmp .tmp

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/server cmd/server/main.go

FROM alpine:latest

WORKDIR /go/bin

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

COPY --from=builder /go/bin/server /go/bin/server
COPY --from=builder /app/.env .env
COPY --from=builder /app/.tmp .tmp

RUN chown appuser:appuser .tmp

USER appuser:appuser

ENTRYPOINT ["./server", "--config", "../../.env"]