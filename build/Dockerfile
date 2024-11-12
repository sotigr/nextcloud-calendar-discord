FROM golang:1.23 AS builder
 
WORKDIR /src
COPY ./ /src
RUN go build -o /bin/app ./cmd
 
FROM alpine:latest
RUN apk add --no-cache ca-certificates
RUN apk add libc6-compat

WORKDIR /app
COPY --from=builder /bin/app /bin/app
RUN chmod +x /bin/app
CMD ["/bin/app"] 