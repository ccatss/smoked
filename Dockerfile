FROM golang:alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o smoked

FROM alpine

RUN apk update && apk add --no-cache bird mtr tcptraceroute
COPY --from=builder /app/smoked /usr/bin/smoked
CMD ["/usr/bin/smoked"]