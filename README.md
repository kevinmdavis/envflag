# Envflag - set flags using environment variables

The Go `flag` package is very useful for parsing command line arguments and configuring program behavior. However, many
platforms encourage the use of the [12 Factor app methodology](http://12factor.net) and provide application configuration using
environment variables rather than command line arguments.

This package makes it possible to bind environment variables to flag values. Your application can continue to use the
standard library `flag` package while also accepting values from environment variables.

## Usage
```go
package main

import (
	"flag"
	"fmt"

	"github.com/kevinmdavis/envflag"
)

var (
	someURL = flag.String("some-url", "", "some url")
)

func main() {
	// envflag.BindAll() must be called before flag.Parse()!
	envflag.BindAll()
	flag.Parse()
	fmt.Println(fmt.Sprintf("URL is %q", *someURL))
}
```

Running:
```bash
$ SOME_URL=www.example.com ./app
URL is "www.example.com"

# Values provided on the comamnd line take precedence.
$ SOME_URL=www.example.com/url1 ./app --some-url=www.example.com/url2
URL is "www.example.com/url2"
```

### Environment Variables Prefixes

Prefixes can be provided in order to support basic namespacing of environment variables. Additionally, "strict" prefixes can
be used to ensure all provided environment variables map to a defined flag (similar to how the `flag` package handles
undefined flags). This is helpful for protecting against typos in environment variable names.

```go
package main

import (
	"flag"
	"fmt"

	"github.com/kevinmdavis/envflag"
)

var (
	port = flag.Int("port", 8080, "the port to listen on")
)

func main() {
	// Environment variables will be prefixed with "MYAPP_".
	envflag.Bind(envflag.NewPrefix("MYAPP", envflag.Strict(true)))
	flag.Parse()
	fmt.Println(fmt.Sprintf("Listening on port: %d", *port))
}
```

```bash
$ MYAPP_PORT=9000 ./app
Listening on port: 9000

$ MYAPP_UNKNOWN=9000 ./app
environment variable provided but corresponding flag not defined: MYAPP_UNKNOWN
Usage of ./app:
  -port int
    	the port to listen on [MYAPP_PORT] (default 8080)


$ MYAPP_PORT=abc ./app
invalid value "abc" for environment variable MYAPP_PORT: parse error
Usage of ./app:
  -port int
    	the port to listen on [MYAPP_PORT] (default 8080)
```

`Bind()` also supports multiple prefixes. The example below would accept values from both environment variables `PORT`
and `MYAPP_PORT`. Since the `MYAPP` prefix is strict, setting an environment variable `MYAPP_UNKNOWN_VAR` will cause the
program to exit. Other environment variables such as `MY_UNRELATED_VAR` will simply be ignored.

```go
envflag.Bind(envflag.AllEnv, envflag.NewPrefix("MYAPP", envflag.Strict(true)))
```

## License

[Apache 2.0](LICENSE)

This is not an officially supported Google product.
