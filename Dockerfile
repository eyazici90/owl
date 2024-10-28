FROM golang:1.23.2-alpine AS build

ENV CGO_ENABLED=0\
    GOOS=linux\
    GOARCH=amd64

WORKDIR /build
COPY . .
RUN go build -o /bin /cmd/owl/main.go

#
FROM gcr.io/distroless/static-debian12
COPY --from=build /build/bin /usr/local/bin/owl
WORKDIR /owl
CMD ["/usr/local/bin/owl"]