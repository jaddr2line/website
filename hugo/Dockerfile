from scratch as unpack
add hugo.tar.gz /hugo

from alpine:3.10 as certs
run apk --no-cache add ca-certificates

from scratch
copy --from=unpack /hugo/hugo /bin/hugo
copy --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
workdir /site
entrypoint ["/bin/hugo", "server", "--disableLiveReload", "--renderToDisk", "--bind", "0.0.0.0"]
