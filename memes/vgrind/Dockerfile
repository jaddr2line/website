from golang:1.12.6-alpine3.10 as builder
run apk --no-cache add ca-certificates
copy main.go /go/src/github.com/jadr2ddude/website/vgrind/
run CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install -v github.com/jadr2ddude/website/vgrind

from alpine:3.10.0
run apk --no-cache add docker
copy --from=builder /go/bin/vgrind /bin/vgrind
copy --from=builder /usr/share/ca-certificates /usr/share/ca-certificates
add ./v /root/v
add ./index.html /root/v/static/
workdir /root/v
entrypoint ["/bin/vgrind"]
