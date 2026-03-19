# Contributing

Hi! Thanks for taking an interest in this project; we appreciate all
contributions, no matter how big or small. So that your experience as a
contributor is as frictionless as possible, please make sure to:

1. **Write commit messages according to the [Conventional
   Commits](https://www.conventionalcommits.org/en/v1.0.0/) system.** For the
   impatient, this means that your commits should look something like:
   ```text
   fix: prevent racing of requests

   Introduce a request ID and a reference to latest request so that responses
   other than from the latest request can be dismissed. Also, remove timeouts
   which were used to mitigate the racing issue but are obsolete now.
   ```
   Note that the summary line includes a prefix with the type of change that is
   being made (other common types are `feat:`, `refactor:`, `docs:`, and
   `chore:`) and is written in the imperative tense with no full stop. The
   commit body then expands on the summary and should explain the motivation
   behind the change.
2. **Discuss substantial changes in an issue before submitting a pull request.**
   Doing so gives us a chance to determine whether we believe the proposed
   changes are something that belongs in the project and avoids us potentially
   rejecting a pull request you have put a lot of effort into, which feels bad
   for both parties involved. For bug fixes, typo corrections, documentation
   updates, small features, and so on, feel free to fire away!
3. **Follow the [libdns package
   requirements](https://github.com/libdns/libdns/wiki/Implementing-a-libdns-package#requirements).**

## Prerequisites

This project only requires that you have a working [Go](https://go.dev/dl/)
toolchain installed. If you're on a Linux or macOS system, your package manager
(eg. apt, pacman, or homebrew) likely already has a `go` package that you can
install.

## Building

Build this project with `go build` to verify that the code compiles. Because
this is a library, no binary will be produced.

## Testing 

Run unit tests with `go test`. For verbose output that lists every test that has
been run, add the `-v` flag: `go test -v`.

## Using a local copy in another project

If you are working on this library and want to test your changes in a consuming
project (e.g. a Caddy plugin or a standalone tool), use a
[`replace` directive](https://go.dev/ref/mod#go-mod-file-replace) in the
consuming project's `go.mod`:

```bash
go mod edit -replace github.com/sitehostnz/libdns-sitehost=/path/to/libdns-sitehost
```

This adds a line like:

```text
replace github.com/sitehostnz/libdns-sitehost => /path/to/libdns-sitehost
```

You can then import and use the library as normal. For example, create a test
file:

```go
package main

import (
	"context"
	"fmt"
	"os"

	sitehost "github.com/sitehostnz/libdns-sitehost"
)

func main() {
	provider := &sitehost.Provider{
		ClientID: os.Getenv("SITEHOST_CLIENT_ID"),
		APIKey:   os.Getenv("SITEHOST_API_KEY"),
	}

	records, err := provider.GetRecords(context.Background(), "example.com.")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	for _, r := range records {
		fmt.Println(r.RR())
	}
}
```

Run it with:

```bash
SITEHOST_CLIENT_ID=your-client-id SITEHOST_API_KEY=your-api-key go run main.go
```

Once you are done testing, remove the replace directive:

```bash
go mod edit -dropreplace github.com/sitehostnz/libdns-sitehost
```

## Licence

By contributing to this project, you agree that your contributions will be
licensed under the [MIT Licence](LICENSE).
