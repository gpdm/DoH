# Accept the Go version for the image to be set as a build argument.
# Default to Go 1.13
ARG GO_VERSION=1.13

# support dockerhub dynamic build paramters
ARG BUILD_DATE
ARG VCS_REF

# First stage: build the executable.
FROM golang:${GO_VERSION}-alpine AS builder

LABEL \
    org.label-schema.build-date=$BUILD_DATE \
    org.label-schema.schema-version="1.0" \
    org.label-schema.vendor="Gianpaolo Del Matto <buildmaint@phunsites.net>" \
    org.label-schema.description="This is a 'DNS over HTTP' (DoH) server implementation written in Go." \
    org.label-schema.name=DoH \
    org.label-schema.vcs-ref=$VCS_REF \
    org.label-schema.vcs-url=https://github.com/gpdm/DoH

# create config dir
RUN mkdir /conf

# Install the Certificate-Authority certificates for the app to be able to make
# calls to HTTPS endpoints.
# Git is required for fetching the dependencies.
RUN apk add --no-cache ca-certificates git tzdata zip

# Set the environment variables for the go command:
# * CGO_ENABLED=0 to build a statically-linked executable
ENV CGO_ENABLED=0

# Set up the time zone data so go can use time.Location
WORKDIR /usr/share/zoneinfo
# -0 means no compression.  Needed because go's
# tz loader doesn't handle compressed data.
RUN zip -r -0 /zoneinfo.zip .

# Set the working directory outside $GOPATH to enable the support for modules.
WORKDIR /src

# Fetch dependencies first; they are less susceptible to change on every build
# and will therefore be cached for speeding up the next build
COPY ./go.mod ./go.sum bootstrap/main.go ./
RUN go mod download
RUN time go build -installsuffix 'static'
RUN rm main.go

# Import the code from the context.
COPY ./ ./

ARG APP_VERSION=unversioned

RUN export | grep -i app_version

# Build a static executable to `/DoH`
RUN go build \
    -installsuffix 'static' \
    -o /DoH

# Final stage: the running container.
FROM scratch AS final

# Import the user and group files from the first stage.
COPY --from=builder /etc/group /etc/passwd /etc/

# Import the Certificate-Authority certificates for enabling HTTPS.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Import the compiled executable from the first stage.
COPY --from=builder /DoH /DoH

# Import the time zone data
COPY --from=builder /zoneinfo.zip /

# add configuration directory
COPY --from=builder /conf /conf

# Tell go where to finde the time zone data
ENV ZONEINFO /zoneinfo.zip \
    GLOBAL.LISTEN= \
    GLOBAL.LOGLEVEL=5 \
    HTTP.ENABLE=0 \
    HTTP.PORT=80 \
    TLS.ENABLE=1 \
    TLS.PORT=443 \
    TLS.PKEY=./conf/private.key \
    TLS.CERT=./conf/public.crt \
    DNS.RESOLVERS= \
    REDIS.ENABLE=0 \
    REDIS.ADDR= \
    REDIS.PORT=6379 \
    REDIS.USERNAME= \
    REDIS.PASSWORD= \
    INFLUX.ENABLE=0 \
    INFLUX.URL= \
    INFLUX.DATABASE= \
    INFLUX.USERNAME= \
    INFLUX.PASSWORD=

# Declare the port on which the webserver will be exposed.
# As we're going to run the executable as an unprivileged user, we can't bind
# to ports below 1024.
EXPOSE 8080
EXPOSE 8443

# Perform any further action as an unprivileged user.
USER nobody:nobody

# Run the compiled binary.
ENTRYPOINT ["/DoH"]