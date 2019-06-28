package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"
)

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

func buildContainer(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "build", "-t", "jadendw.dev/vgrind", ".")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	go func() {
		<-c
		cancel()
	}()
	signal.Notify(c, os.Interrupt)

	var status uint32
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		switch atomic.LoadUint32(&status) {
		case 0:
			w.Write([]byte("starting"))
		case 1:
			w.Write([]byte("ready"))
		}
	})
	var lck sync.Mutex
	http.HandleFunc("/vgrind", func(w http.ResponseWriter, r *http.Request) {
		lck.Lock() //limit to one concurrent op
		defer lck.Unlock()

		if atomic.LoadUint32(&status) != 1 {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), time.Minute)
		defer cancel()

		cmd := exec.CommandContext(ctx, "docker", "run", "-m", "100m", "--cpus=1", "--rm", "-i", "jadendw.dev/vgrind")
		cmd.Stdin = io.LimitReader(r.Body, 1*1024*1024)
		cmd.Stdout = w
		cmd.Stderr = w
		err := cmd.Run()
		if err != nil {
			io.WriteString(w, err.Error())
		}
	})

	srv := http.Server{}

	var wg sync.WaitGroup
	defer wg.Wait()
	serverCtx(ctx, &wg, &srv)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		log.Println("starting http server")
		err := srv.ListenAndServe()
		if err != nil && ctx.Err() == nil {
			log.Printf("http server failed: %s", err)
		}
	}()

	log.Println("building container image")
	buildContainer(ctx)
	log.Println("done building image")
	atomic.StoreUint32(&status, 1)
	log.Println("ready")

	<-ctx.Done()
	log.Println("goodbye")
}
