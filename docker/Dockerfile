FROM golang:1.26 AS builder

WORKDIR /app

COPY ./go.mod ./go.sum  ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/algo-bot ./cmd/bot

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /out/algo-bot /algo-bot
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/algo-bot"]
