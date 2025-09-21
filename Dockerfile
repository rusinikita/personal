FROM alpine
# To load timezone from TZ env
RUN apk update && apk add --no-cache tzdata
# Copy our static executable.
COPY ./build/app /go/bin/app
# Run the app binary.
ENTRYPOINT ["/go/bin/app"]