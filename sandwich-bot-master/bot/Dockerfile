FROM golang:1.20 as builder

LABEL maintainer="github.com/sol-3made"

WORKDIR /bot

COPY go.mod .
COPY go.sum .
RUN go mod download && go mod verify
RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY . .
RUN swag init

RUN CGO_ENABLED=0 GOOS=linux go build -a -o bot .
RUN chmod +x bot
#
#
FROM alpine:latest as production

RUN apk --no-cache add ca-certificates bash vim curl libc6-compat

WORKDIR /usr/src/app/

COPY --from=builder /bot/bot .
COPY --from=builder /bot/docs .
COPY --from=builder /bot/nodes.json .
COPY --from=builder /bot/abi ./abi

CMD ["./bot"]