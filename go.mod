module github.com/jakubkulhan/fcgipass

require (
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/pkg/errors v0.8.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/tomasen/fcgi_client v0.0.0-20180423082037-2bb3d819fd19
)

replace github.com/tomasen/fcgi_client v0.0.0-20180423082037-2bb3d819fd19 => github.com/jakubkulhan/fcgi_client v0.0.0-20181129091851-a641692733ac34c81b45ddde03381c6e2f086083
