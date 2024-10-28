ARG ARCH="amd64"
ARG OS="linux"

FROM golang:1.23.2-alpine AS build

ENV CGO_ENABLED=0\
    GOOS=${OS}\
    GOARCH=${ARCH}

WORKDIR /build
COPY . .
RUN go build -o /bin/owl ./cmd/owl

#
FROM gcr.io/distroless/static-debian12
COPY --from=build /bin/owl /usr/local/bin/owl
ENTRYPOINT ["/usr/local/bin/owl"]