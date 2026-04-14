module github.com/sitehostnz/libdns-sitehost/libdnstest

go 1.26.11

require (
	github.com/libdns/libdns v1.2.0-alpha.1
	github.com/sitehostnz/libdns-sitehost v0.0.0
)

require (
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/sitehostnz/gosh v0.7.2-0.20260911023802-f683e8b399f4 // indirect
)

replace github.com/sitehostnz/libdns-sitehost => ../
