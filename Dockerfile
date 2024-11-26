# syntax=docker/dockerfile:1

FROM golang:1.23-alpine

RUN apk update && apk upgrade
RUN apk add chromium font-noto font-noto-emoji

WORKDIR /app

COPY go.mod .
RUN go mod download

COPY . .

RUN go build -o /main
EXPOSE 3000
CMD [ "/main" ]
