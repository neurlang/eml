FROM golang:latest-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/emlserver/ ./cmd/emlserver/

RUN cd emlui/ && GOOS=js GOARCH=wasm go build -o emlui.wasm && \
    cd .. && go build -o emlserver .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/cmd/emlserver/emlserver .
COPY --from=builder /app/cmd/emlserver/emlui/ ./emlui/

EXPOSE 8080

CMD ["./emlserver"]
