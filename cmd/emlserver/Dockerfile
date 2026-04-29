FROM golang:alpine AS builder

WORKDIR /app

RUN go mod init emlserver
RUN go mod download
RUN go get github.com/neurlang/eml/regression

RUN cd emlui/ && GOOS=js GOARCH=wasm go build -o emlui.wasm && \
    cd .. && go build -o emlserver .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/emlserver .

EXPOSE 8080

CMD ["./emlserver"]
