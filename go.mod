module github.com/sitehostnz/libdns-sitehost

go 1.26.11

require (
	github.com/libdns/libdns v1.1.1
	github.com/sitehostnz/gosh v0.7.2-0.20260911023802-f683e8b399f4 // TODO: replace with the first tagged release that includes gosh PR #65 (context cancellation fix) before merging.
)

require github.com/google/go-querystring v1.1.0 // indirect
