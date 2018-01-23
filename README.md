# fcgipass

> Proxy HTTP requests to FastCGI server

## Usage

```
Usage of fcgipass:
  -dial-fcgi <network>:<address>
    	in format <network>:<address>, e.g. tcp:127.0.0.1:9000
  -document-root directory
    	directory that will be prepended to request path to be passed as SCRIPT_FILENAME
  -listen-http address
    	TCP address to listen on, e.g. 0.0.0.0:80
  -script-filename SCRIPT_FILENAME
    	passed to FastCGI process as SCRIPT_FILENAME, overrides document root
```

`fcgipass` will listen on port `8000` and route requests to FastCGI server on port `9000`:

```sh
$ fcgipass -listen-http :8000 -dial-fcgi tcp:127.0.0.1:9000 -document-root /var/www
```

If all requests should be routed to the same file of FastCGI server, specify `-script-filename`:

```sh
$ fcgipass -listen-http :8000 -dial-fcgi tcp:127.0.0.1:9000 -script-filename /var/www/index.php
```

## License

Licensed under MIT license. See `LICENSE` file.
