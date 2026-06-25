# Use the latest version of Go as the base image
FROM golang:1.23 AS base

# Install needed dependencies for base image and update certs
RUN apt-get update \
  && apt-get install -y make openssh-client ca-certificates unzip \
    && update-ca-certificates

# Download proto zip
ENV PROTOC_ZIP=protoc-24.0-linux-x86_64.zip
RUN curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v24.0/${PROTOC_ZIP}
RUN unzip -o ${PROTOC_ZIP} -d ./proto
RUN chmod 755 -R ./proto/bin
ENV BASE=/usr
# Copy into path
RUN cp ./proto/bin/protoc ${BASE}/bin/
RUN cp -R ./proto/include/* ${BASE}/include/

# Download protoc-gen-grpc-web
ENV GRPC_WEB=protoc-gen-grpc-web-1.5.0-linux-x86_64
ENV GRPC_WEB_PATH=/usr/bin/protoc-gen-grpc-web
RUN curl -OL https://github.com/grpc/grpc-web/releases/download/1.5.0/${GRPC_WEB}
# Copy into path
RUN mv ${GRPC_WEB} ${GRPC_WEB_PATH}
RUN chmod +x ${GRPC_WEB_PATH}

RUN go env -w GO111MODULE=on
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.0
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0


# create a build artifact
FROM base AS builder
# Set the working directory to the root of the project
WORKDIR /app

# Copy the Go dependencies file and download the dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the Makefile and the rest of the source code
COPY Makefile .
COPY protocols ./protocols
# NOTE: vendored into the lia repo without its own .git, so the original
# `COPY .git ./.git` is dropped. The Makefile's git-derived version ldflags
# resolve to empty strings (non-fatal); proto generation + build don't need git.
COPY . ./

RUN make generate-proto

# Build the application
RUN make build

# Create a new, smaller image based on the scratch image (an empty, executable image)
FROM scratch

# Copy the SSL certificates from the base image
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the built executable from the builder image
COPY --from=builder /app/gateguard /gateguard

# Set the entrypoint to the executable
ENTRYPOINT ["/gateguard", "serve"]
