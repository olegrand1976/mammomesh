module github.com/olegrand1976/mammomesh/backend

go 1.27.0

require (
	github.com/go-chi/chi/v5 v5.5.0
	github.com/jackc/pgx/v5 v5.11.0
	github.com/modelcontextprotocol/go-sdk v1.8.0
	github.com/redis/go-redis/v9 v9.23.0
	golang.org/x/crypto v0.58.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.3 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.uber.org/atomic v1.12.0 // indirect
	golang.org/x/oauth2 v0.37.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/time v0.16.0 // indirect
)

// Les modules golang.org/x, go.uber.org et go.yaml.in sont pris sur leurs dépôts GitHub officiels :
// même code, accessible depuis les environnements où golang.org est filtré.
replace (
	go.uber.org/atomic => github.com/uber-go/atomic v1.12.0
	go.yaml.in/yaml/v3 => github.com/yaml/go-yaml/v3 v3.0.5
	golang.org/x/crypto => github.com/golang/crypto v0.58.0
	golang.org/x/mod => github.com/golang/mod v0.42.0
	golang.org/x/net => github.com/golang/net v0.61.0
	golang.org/x/oauth2 => github.com/golang/oauth2 v0.37.0
	golang.org/x/sync => github.com/golang/sync v0.24.0
	golang.org/x/sys => github.com/golang/sys v0.49.0
	golang.org/x/text => github.com/golang/text v0.43.0
	golang.org/x/time => github.com/golang/time v0.16.0
	golang.org/x/tools => github.com/golang/tools v0.52.0
	gopkg.in/check.v1 => github.com/go-check/check v0.0.0-20201130134442-10cb98267c6c
	gopkg.in/yaml.v3 => github.com/go-yaml/yaml/v3 v3.0.1
)
