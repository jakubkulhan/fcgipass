# fcgipass

> Proxy HTTP requests to FastCGI server

## Usage

```
Proxy HTTP requests to FastCGI server

Usage:
  fcgipass [flags]

Flags:
  -d, --address string           FastCGI server address. (default "localhost:9000")
  -r, --document-root string     Document root will be prepended to request path to be passed as SCRIPT_FILENAME. (default ".")
      --health string            Path the server won't route to backend FastCGI server, but respond with 200 OK instead (for health checks). (default "/healthz")
  -h, --help                     help for fcgipass
  -b, --host string              Bind HTTP listener to this host. If not specified, listens on all interfaces.
  -n, --network string           FastCGI server network. (default "tcp")
  -p, --port int                 Listen for HTTP requests on this port. (default 80)
  -f, --script-filename string   Passed to FastCGI as SCRIPT_FILENAME, overrides document root.
  -s, --socket string            Listen for HTTP requests on this UNIX-domain socket.
```

`fcgipass` will listen on port `8000` and route requests to FastCGI server on port `9000`:

```sh
$ fcgipass -p 8000 -d 127.0.0.1:9000 --document-root /var/www
```

If all requests should be routed to the same file of FastCGI server, specify `--script-filename`:

```sh
$ fcgipass -p 8000 -d 127.0.0.1:9000 --script-filename /var/www/index.php
```

## License

Licensed under MIT license. See `LICENSE` file.
