# Use the latest version of Go as the base image
FROM golang:1.26 AS base

# Install needed dependencies for base image and update certs
RUN apt-get update \
  && apt-get install -y make openssh-client ca-certificates unzip \
    && update-ca-certificates

# create a build artifact
FROM base AS builder
# Set the working directory to the root of the project
WORKDIR /app

# Copy the Go dependencies file and download the dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the Makefile and the rest of the source code.
# NOTE: backend/ is a subdirectory of the Lia monorepo, so there is no .git in
# the build context. git-derived version ldflags are therefore empty in the
# container build (the Makefile tolerates this). `COPY . ./` still picks up .git
# if this service is ever built from its own repo root.
COPY Makefile .
COPY . ./

# Build the application
RUN make build

# Create a new, smaller image based on the scratch image (an empty, executable image)
FROM scratch

# Copy the SSL certificates from the base image
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the built executable from the builder image
COPY --from=builder /app/lia /lia

# Set the entrypoint to the executable
ENTRYPOINT ["/lia"]
CMD ["serve"]
