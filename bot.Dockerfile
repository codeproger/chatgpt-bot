# build stage
FROM golang:1.21 AS builder

#ARG GITLAB_LOGIN
#ARG GITLAB_TOKEN

ADD . /src
WORKDIR /src

# Create a "nobody" non-root user for the next image by crafting an /etc/passwd
# file that the next image can copy in. This is necessary since the next image
# is based on scratch, which doesn't have adduser, cat, echo, or even sh.
RUN echo "nobody:x:65534:65534:Nobody:/:" > /etc_passwd

#RUN echo "machine gitlab.afkxxx.com login ${GITLAB_LOGIN} password ${GITLAB_TOKEN}" > ~/.netrc

RUN apt-get update && apt-get install -y \
    libmp3lame-dev \
    libopus-dev \
    opus-tools \
    libopusfile-dev

ARG VERSION="latest"
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64
RUN go build -a -installsuffix cgo -o app

#USER nobody
#RUN #chmod +x app
CMD ["/src/app"]

### final stage
###FROM scratch
FROM debian:12

WORKDIR /bin

###
COPY --from=builder /src/app /bin/app
COPY --from=builder /src/config.json /bin/config.json
COPY --from=builder /src/opus /bin/opus

RUN touch bot.db
RUN chmod 777 bot.db

RUN apt-get update && apt-get install -y \
    libmp3lame-dev \
    libopus-dev \
    opus-tools \
    libopusfile-dev

# for compatibility with CGO_ENABLED=1
RUN apt-get update && apt-get install -y libc6

RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates

##
##RUN apk add opus lame
#### Copy the /etc_passwd file we created in the builder stage into /etc/passwd in
#### the target stage. This creates a new non-root user as a security best
#### practice.
COPY --from=0 /etc_passwd /etc/passwd

###
#USER nobody
CMD ["/bin/app", "/bin/config.json"]
