package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
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
	memeRoute.HandleFunc("/vgrind/", func(w http.ResponseWriter, r *http.Request) {
		vgrindprox.ServeHTTP(w, r)
	})

	srv := http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Host {
			case "jadendw.dev", "www.jadendw.dev":
				huprox.ServeHTTP(w, r)
			case "memes.jadenw.dev":
				memeRoute.ServeHTTP(w, r)
			case "jadendw.com", "www.jadendw.com":
				http.Redirect(w, r, "https://jadendw.dev", http.StatusMovedPermanently)
			default:
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			}
		}),
	}

	var wg sync.WaitGroup
	defer wg.Wait()
	serverCtx(ctx, &wg, &srv)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		err := srv.ListenAndServeTLS("/srv/secrets/proxy.pem", "/srv/secrets/proxy.key")
		if err != nil && ctx.Err() == nil {
			log.Printf("https server failed: %s", err)
		}
	}()

	log.Println("server running")
	<-ctx.Done()
	log.Println("goodbye")
}
