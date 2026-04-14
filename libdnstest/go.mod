module github.com/sitehostnz/libdns-sitehost/libdnstest

go 1.26.2

require (
	github.com/libdns/libdns v1.2.0-alpha.1
	github.com/sitehostnz/libdns-sitehost v0.0.0
)

require (
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/sitehostnz/gosh v0.6.0 // indirect
)

replace github.com/sitehostnz/libdns-sitehost => ../
