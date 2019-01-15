FROM golang:1.11.0-stretch as builder
WORKDIR /go/src/app
COPY . .
RUN go get -d -v ./...
RUN go install -v ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o main .
RUN go get -u github.com/rakyll/hey

FROM centos:latest
RUN yum -y install bind-utils
RUN getent group  engineer &> /dev/null || groupadd -r engineer &> /dev/null && \
    getent passwd engineer &> /dev/null || useradd -u 1001 -r -g 0 -d $HOME -s /sbin/nologin \
    -c 'engineer' engineer &> /dev/null
COPY --from=builder /go/src/app/main /usr/bin/
COPY --from=builder /go/bin/hey /usr/bin/
WORKDIR /usr/bin/
EXPOSE 8080
USER 1001
CMD ["./main"]