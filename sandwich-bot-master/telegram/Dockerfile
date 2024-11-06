FROM golang:1.20-alpine AS builder

LABEL maintainer="github.com/sol-3made"

WORKDIR /usr/src/app

COPY . .

RUN go mod download && go mod verify

RUN go build -v -o telegram_bot
RUN chmod +x telegram_bot

FROM alpine:latest

WORKDIR /usr/src/app

RUN apk --no-cache add ca-certificates bash vim curl libc6-compat

COPY --from=builder /usr/src/app/telegram_bot .
COPY --from=builder /usr/src/app/.env.json .

CMD ["./telegram_bot"]
# CMD ["tail", "-f", "/dev/null"]
