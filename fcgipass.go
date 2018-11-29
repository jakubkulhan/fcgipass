package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tomasen/fcgi_client"
)

type Server struct {
	host                   string
	port                   int
	socket                 string
	documentRoot           string
	scriptFilenameOverride string
	healthCheckPath        string
	network                string
	address                string
}

func main() {
	server := &Server{}

	rootCmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "Proxy HTTP requests to FastCGI server",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if server.address == "" {
				return errors.New("you have to pass FastCGI server address")
			}

			var l net.Listener
			if server.socket != "" {
				l, err = net.Listen("unix", server.socket)
			} else {
				l, err = net.Listen("tcp", fmt.Sprintf("%s:%d", server.host, server.port))
			}
			if err != nil {
				return errors.Wrap(err, "listen failed")
			}
			defer l.Close()

			httpServer := &http.Server{
				Handler: server,
			}

			interrupt := make(chan os.Signal, 1)
			signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-interrupt
				log.Println("gracefully shutting down")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := httpServer.Shutdown(ctx); err != nil {
					log.Println("graceful shutdown failed: ", err)
				} else {
					log.Println("shutdown complete")
				}
			}()

			log.Println("starting HTTP server on", l.Addr())

			if err := httpServer.Serve(l); err == http.ErrServerClosed {
				return nil
			} else {
				return err
			}
		},
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	rootCmd.Flags().StringVarP(&server.host, "host", "b", "", "Bind HTTP listener to this host. If not specified, listens on all interfaces.")
	rootCmd.Flags().IntVarP(&server.port, "port", "p", 80, "Listen for HTTP requests on this port.")
	rootCmd.Flags().StringVarP(&server.socket, "socket", "s", "", "Listen for HTTP requests on this UNIX-domain socket.")
	rootCmd.Flags().StringVarP(&server.documentRoot, "document-root", "r", wd, "Document root will be prepended to request path to be passed as SCRIPT_FILENAME.")
	rootCmd.Flags().StringVarP(&server.scriptFilenameOverride, "script-filename", "f", "", "Passed to FastCGI as SCRIPT_FILENAME, overrides document root.")
	rootCmd.Flags().StringVar(&server.healthCheckPath, "health", "/healthz", "Path the server won't route to backend FastCGI server, but response with 200 OK (for health checks).")
	rootCmd.Flags().StringVarP(&server.network, "network", "n", "tcp", "FastCGI server network.")
	rootCmd.Flags().StringVarP(&server.address, "address", "d", "", "FastCGI server address.")

	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	if s.healthCheckPath != "" && request.URL.Path == s.healthCheckPath {
		w.WriteHeader(200)
		if _, err := w.Write([]byte("ok\n")); err != nil {
			log.Println("health write failed:", err)
		}
		return
	}

	client, err := fcgiclient.DialContext(request.Context(), s.network, s.address)
	if err != nil {
		log.Println("dial failed:", err)
		http.Error(w, "Cannot dial backend", 502)
		return
	}

	defer client.Close()
	defer request.Body.Close()

	scriptFilename := path.Join(s.documentRoot, request.URL.Path)
	if s.scriptFilenameOverride != "" {
		scriptFilename = s.scriptFilenameOverride
	}

	remoteAddr, remotePort, _ := net.SplitHostPort(request.RemoteAddr)
	request.URL.Path = request.URL.ResolveReference(request.URL).Path
	env := map[string]string{
		"CONTENT_LENGTH":  fmt.Sprintf("%d", request.ContentLength),
		"CONTENT_TYPE":    request.Header.Get("Content-Type"),
		"FCGI_ADDR":       s.address,
		"FCGI_PROTOCOL":   s.network,
		"HTTP_HOST":       request.Host,
		"PATH_INFO":       request.URL.Path,
		"QUERY_STRING":    request.URL.Query().Encode(),
		"REMOTE_ADDR":     remoteAddr,
		"REMOTE_PORT":     remotePort,
		"REQUEST_METHOD":  request.Method,
		"REQUEST_PATH":    request.URL.Path,
		"REQUEST_URI":     request.URL.RequestURI(),
		"SCRIPT_FILENAME": scriptFilename,
		"SERVER_NAME":     request.Host,
		"SERVER_PROTOCOL": request.Proto,
		"SERVER_SOFTWARE": os.Args[0],
	}

	if s.socket == "" {
		if s.host == "" {
			env["SERVER_ADDR"] = request.Host
		} else {
			env["SERVER_ADDR"] = s.host
		}
		env["SERVER_PORT"] = strconv.Itoa(s.port)
	}

	for k, v := range request.Header {
		env["HTTP_"+strings.Replace(strings.ToUpper(k), "-", "_", -1)] = strings.Join(v, ";")
	}

	response, err := client.Request(env, request.Body)
	if err != nil {
		log.Println("request failed:", err)
		http.Error(w, "Unable to fetch response from backend", 502)
		return
	}

	response.Status = response.Header.Get("Status")
	response.StatusCode, _ = strconv.Atoi(strings.Split(response.Status, " ")[0])
	if response.StatusCode < 100 {
		response.StatusCode = 200
	}
	defer response.Body.Close()

	for k, v := range response.Header {
		for i := 0; i < len(v); i++ {
			if w.Header().Get(k) == "" {
				w.Header().Set(k, v[i])
			} else {
				w.Header().Add(k, v[i])
			}
		}
	}

	w.WriteHeader(response.StatusCode)

	var written int64
	if request.Method != "HEAD" {
		written, _ = io.Copy(w, response.Body)
	}

	log.Printf(
		`"%s %s %s" %d %d "%s"`,
		request.Method,
		request.URL.Path,
		request.Proto,
		response.StatusCode,
		written,
		request.Header.Get("User-Agent"),
	)
}
