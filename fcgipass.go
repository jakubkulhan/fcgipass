package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/tomasen/fcgi_client"
)

var (
	listenHttp             = flag.String("listen-http", "", "TCP `address` to listen on, e.g. 0.0.0.0:80")
	dialFcgi               = flag.String("dial-fcgi", "", "in format `<network>:<address>`, e.g. tcp:127.0.0.1:9000")
	documentRoot           = flag.String("document-root", "", "`directory` that will be prepended to request path to be passed as SCRIPT_FILENAME")
	scriptFilenameOverride = flag.String("script-filename", "", "passed to FastCGI process as `SCRIPT_FILENAME`, overrides document root")
)

var (
	serverSoftware string
	serverAddr     string
	serverPort     string
	fcgiAddr       string
	fcgiProto      string
)

func main() {
	flag.Parse()

	if *listenHttp == "" || *dialFcgi == "" || (*scriptFilenameOverride == "" && *documentRoot == "") {
		flag.Usage()
		os.Exit(1)
		return
	}

	serverSoftware = os.Args[0]
	serverAddr = *listenHttp
	fcgiAddr = *dialFcgi

	if strings.HasPrefix(serverAddr, ":") {
		serverPort = serverAddr[1:]
		serverAddr = "0.0.0.0"
	} else if parts := strings.SplitN(serverAddr, ":", 2); len(parts) == 2 {
		serverAddr = parts[0]
		serverPort = parts[1]
	} else {
		flag.Usage()
		os.Exit(1)
		return
	}

	if parts := strings.SplitN(fcgiAddr, ":", 2); len(parts) == 2 {
		fcgiProto = parts[0]
		fcgiAddr = parts[1]
	} else {
		flag.Usage()
		os.Exit(1)
		return
	}

	if err := http.ListenAndServe(serverAddr+":"+serverPort, http.HandlerFunc(serve)); err != nil {
		log.Fatal(err)
		os.Exit(1)
		return
	}

	os.Exit(0)
}

func serve(w http.ResponseWriter, r *http.Request) {
	client, err := fcgiclient.Dial(fcgiProto, fcgiAddr)
	if err != nil {
		log.Println(err)
		http.Error(w, "502 Bad Gateway", 502)
		return
	}

	defer client.Close()
	defer r.Body.Close()

	scriptFilename := path.Join(*documentRoot, r.URL.Path)
	if *scriptFilenameOverride != "" {
		scriptFilename = *scriptFilenameOverride
	}

	remoteAddr, remotePort, _ := net.SplitHostPort(r.RemoteAddr)
	r.URL.Path = r.URL.ResolveReference(r.URL).Path
	env := map[string]string{
		"CONTENT_LENGTH":  fmt.Sprintf("%d", r.ContentLength),
		"CONTENT_TYPE":    r.Header.Get("Content-Type"),
		"FCGI_ADDR":       fcgiAddr,
		"FCGI_PROTOCOL":   fcgiProto,
		"HTTP_HOST":       r.Host,
		"PATH_INFO":       r.URL.Path,
		"QUERY_STRING":    r.URL.Query().Encode(),
		"REMOTE_ADDR":     remoteAddr,
		"REMOTE_PORT":     remotePort,
		"REQUEST_METHOD":  r.Method,
		"REQUEST_PATH":    r.URL.Path,
		"REQUEST_URI":     r.URL.RequestURI(),
		"SCRIPT_FILENAME": scriptFilename,
		"SERVER_ADDR":     serverAddr,
		"SERVER_NAME":     r.Host,
		"SERVER_PORT":     serverPort,
		"SERVER_PROTOCOL": r.Proto,
		"SERVER_SOFTWARE": serverSoftware,
	}

	for k, v := range r.Header {
		env["HTTP_"+strings.Replace(strings.ToUpper(k), "-", "_", -1)] = strings.Join(v, ";")
	}

	response, err := client.Request(env, r.Body)
	if err != nil {
		log.Println("err> ", err.Error())
		http.Error(w, "Unable to fetch the response from the backend", 502)
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
	if r.Method != "HEAD" {
		written, _ = io.Copy(w, response.Body)
	}

	log.Printf(
		`"%s %s %s" %d %d "%s"`,
		r.Method,
		r.URL.Path,
		r.Proto,
		response.StatusCode,
		written,
		r.Header.Get("User-Agent"),
	)
}
