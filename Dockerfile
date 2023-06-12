FROM golang:1.19-alpine as base_build
ENV CGO_ENABLED=0

RUN mkdir /app
WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY .. .

RUN go build -o ./cmd/main ./cmd/main.go

CMD ["/app/cmd/main"]

# FROM scratch
# COPY --from=base_build ./app .
# CMD ["./compiled"]
