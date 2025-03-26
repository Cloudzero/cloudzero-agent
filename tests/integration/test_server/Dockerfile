FROM golang:1.24.1-alpine

WORKDIR /app

COPY . .

RUN go build -o main .

ARG EXPOSED_PORT=8081
EXPOSE ${EXPOSED_PORT}

CMD ["./main"]
