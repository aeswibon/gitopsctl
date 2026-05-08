# Build stage is handled by GoReleaser
FROM alpine:3.21

RUN apk add --no-cache ca-certificates git openssh tzdata

# Create a non-root user
RUN addgroup -S gitopsctl && adduser -S gitopsctl -G gitopsctl
USER gitopsctl

WORKDIR /app

# GoReleaser will copy the built binary into the image
COPY gitopsctl /usr/local/bin/gitopsctl

ENTRYPOINT ["/usr/local/bin/gitopsctl"]
CMD ["start"]
