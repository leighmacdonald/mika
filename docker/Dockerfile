FROM golang:1.14-alpine
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"

# Set the Current Working Directory inside the container
WORKDIR /mika

RUN apk add make

# Copy go mod and sum files
COPY go.mod go.sum ./
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download
COPY . .
RUN make

EXPOSE 34001
EXPOSE 34000

# Command to run the executable
CMD ["./mika", "serve"]
