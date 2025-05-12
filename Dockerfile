FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /changelog-generator ./main.go

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /changelog-generator /usr/local/bin/changelog-generator
ENTRYPOINT ["/usr/local/bin/changelog-generator"]
