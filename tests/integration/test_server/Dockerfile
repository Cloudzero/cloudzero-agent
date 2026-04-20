FROM golang:1.26.2

WORKDIR /app

COPY . .

RUN go build -o main .

ARG EXPOSED_PORT=8081
EXPOSE ${EXPOSED_PORT}

CMD ["./main"]
