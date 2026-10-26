module github.com/olegrand1976/mammomesh/backend

go 1.24.0

require (
	github.com/go-chi/chi/v5 v5.2.3
	github.com/jackc/pgx/v5 v5.8.0
	github.com/modelcontextprotocol/go-sdk v1.4.0
	github.com/redis/go-redis/v9 v9.7.3
	golang.org/x/crypto v0.47.0
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.3 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.34.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
)

// Les modules golang.org/x sont pris sur leurs miroirs GitHub officiels :
// même code, accessible depuis les environnements où golang.org est filtré.
replace (
	golang.org/x/crypto => github.com/golang/crypto v0.47.0
	golang.org/x/mod => github.com/golang/mod v0.32.0
	golang.org/x/net => github.com/golang/net v0.49.0
	golang.org/x/oauth2 => github.com/golang/oauth2 v0.34.0
	golang.org/x/sync => github.com/golang/sync v0.19.0
	golang.org/x/sys => github.com/golang/sys v0.40.0
	golang.org/x/text => github.com/golang/text v0.33.0
	golang.org/x/tools => github.com/golang/tools v0.41.0
	gopkg.in/check.v1 => github.com/go-check/check v0.0.0-20201130134442-10cb98267c6c
	gopkg.in/yaml.v3 => github.com/go-yaml/yaml/v3 v3.0.1
)
