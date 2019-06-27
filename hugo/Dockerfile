from scratch as unpack
add hugo.tar.gz /hugo

from scratch
copy --from=unpack /hugo/hugo /bin/hugo
workdir /site
entrypoint ["/bin/hugo", "server", "--disableLiveReload", "--renderToDisk", "--bind", "0.0.0.0"]
