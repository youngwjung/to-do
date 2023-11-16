FROM golang:1.20-alpine as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN go build -o /api

FROM alpine
RUN mkdir /app
WORKDIR /app
COPY --from=builder /api /app/api
EXPOSE 9000
CMD ["/app/api"]
