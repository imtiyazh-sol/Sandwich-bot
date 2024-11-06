FROM golang:1.20 as builder

LABEL maintainer="github.com/sol-3made"

WORKDIR /auth

COPY go.* .
# COPY go.sum .
RUN go mod download && go mod verify
RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY . .
RUN swag init

RUN CGO_ENABLED=0 GOOS=linux go build -a -o auth .
RUN chmod +x auth
#
FROM alpine:latest as production

RUN apk --no-cache add ca-certificates bash vim curl libc6-compat

WORKDIR /usr/src/app/

COPY --from=builder /auth/auth .
COPY --from=builder /auth/docs .

CMD ["./auth"]