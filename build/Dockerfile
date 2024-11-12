FROM golang:1.23 AS builder
 
WORKDIR /src
COPY ./ /src
RUN go build -o /bin/app ./cmd
 
FROM golang:1.23-bookworm 

WORKDIR /app
COPY --from=builder /bin/app /bin/app
RUN chmod +x /bin/app
CMD ["/bin/app"] 