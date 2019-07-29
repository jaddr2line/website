package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func forceParse(str string) *url.URL {
	u, err := url.Parse(str)
	if err != nil {
		panic(err)
	}
	return u
}

func serverCtx(ctx context.Context, wg *sync.WaitGroup, srv *http.Server) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		defer srv.Close()

		<-ctx.Done()
		sctx, scancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer scancel()
		srv.Shutdown(sctx)
	}()
}

func shouldEnableHTTPS() (bool, error) {
	_, err := os.Stat("/srv/secrets/proxy.pem")
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func splitFirst(path string) (first string, rest string) {
	path = filepath.Clean(path)
	path = strings.TrimPrefix(path, "/")
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	for i, b := range []byte(path) {
		//intentionally using byte not rune for consistency with stdlib
		if b == '/' {
			return path[:i], path[i:]
		}
	}
	return "", rest
}

func main() {
	huprox := httputil.NewSingleHostReverseProxy(forceParse("http://hugo:1313/"))
	vgrindprox := httputil.NewSingleHostReverseProxy(forceParse("http://vgrind/"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	go func() {
		<-c
		cancel()
	}()
	signal.Notify(c, os.Interrupt)

	memeRoute := http.NewServeMux()
	memeRoute.Handle("/", http.FileServer(http.Dir("memes")))
	memeRoute.Handle("/vgrind/", http.StripPrefix("/vgrind/", vgrindprox))

	hfunc := func(w http.ResponseWriter, r *http.Request) {
		switch r.Host {
		case "jadendw.dev", "www.jadendw.dev":
			huprox.ServeHTTP(w, r)
		case "memes.jadendw.dev":
			memeRoute.ServeHTTP(w, r)
		case "jadendw.com", "www.jadendw.com":
			http.Redirect(w, r, "https://jadendw.dev", http.StatusMovedPermanently)
		default:
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}
	}

	https, err := shouldEnableHTTPS()
	if err != nil {
		panic(err)
	}

	if !https {
		ohfunc := hfunc
		hfunc = func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.Host, ".localhost") {
				http.Redirect(w, r, "http://jadendw.dev.localhost/", http.StatusTemporaryRedirect)
				return
			}
			r.Host = strings.TrimSuffix(r.Host, ".localhost")
			ohfunc(w, r)
		}
	}

	srv := http.Server{
		Handler: http.HandlerFunc(hfunc),
	}

	var wg sync.WaitGroup
	defer wg.Wait()
	serverCtx(ctx, &wg, &srv)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		if https {
			log.Println("enabling https. . . ")
			err := srv.ListenAndServeTLS("/srv/secrets/proxy.pem", "/srv/secrets/proxy.key")
			if err != nil && ctx.Err() == nil {
				log.Printf("https server failed: %s", err)
			}
		} else {
			log.Println("enabling http. . . ")
			err := srv.ListenAndServe()
			if err != nil && ctx.Err() == nil {
				log.Printf("http server failed: %s", err)
			}
		}
	}()

	log.Println("server running")
	<-ctx.Done()
	log.Println("goodbye")
}
