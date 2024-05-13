FROM golang:1.22-alpine as builder

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

WORKDIR src/app/

COPY go.mod .
COPY go.sum .
RUN go mod download
RUN go mod verify

COPY cmd cmd
COPY internal internal
COPY pkg pkg
COPY .env .env
COPY .storage .storage

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/server cmd/server/main.go

FROM scratch

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

COPY --from=builder /go/bin/server /go/bin/server

WORKDIR /go/bin

USER appuser:appuser

ENTRYPOINT ["/go/bin/server"]