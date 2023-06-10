FROM golang:1.17-alpine AS builder

WORKDIR /app

COPY . .
RUN go get github.com/lib/pq
RUN go mod tidy
RUN go build -o app .


FROM alpine:latest


WORKDIR /app


COPY --from=builder /app/app .

EXPOSE 5000

CMD ["./app"]
