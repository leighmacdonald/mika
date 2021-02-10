FROM golang:1.14-alpine as build
ENV CGO_ENABLED 0
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
WORKDIR /build
RUN apk add make git
COPY go.mod go.sum ./
# Download all dependencies. Dependencies will be cached if the
# go.mod and go.sum files are not changed
RUN go mod download
COPY docker .
RUN make

FROM alpine:latest
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
EXPOSE 34001
EXPOSE 34000
RUN apk add bash
WORKDIR /app
VOLUME ["/app/geo_data"]
COPY docker/docker_init.sh .
COPY --from=build /build/mika .
# Command to run the executable
RUN ./mika -v
ENTRYPOINT ["./docker_init.sh"]
CMD ["./mika", "serve"]
