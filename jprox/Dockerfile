from golang:1.12.6-alpine3.10 as builder
run apk --no-cache add ca-certificates
copy main.go /go/src/github.com/jadr2ddude/website/jprox/
run GO111MODULE=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install -v github.com/jadr2ddude/website/jprox

from scratch
copy --from=builder /go/bin/jprox /bin/jprox
copy --from=builder /usr/share/ca-certificates /usr/share/ca-certificates
workdir /root
entrypoint ["/bin/jprox"]
