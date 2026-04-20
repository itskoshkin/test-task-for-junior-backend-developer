module example.com/taskservice

go 1.23.0

require (
	github.com/gorilla/mux v1.8.1
	github.com/jackc/pgx/v5 v5.7.6
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	// ^ CVEs (GO-2025-4116/4134/4135) not closed since version bump requires upgrading to Go 1.24, and pgx uses x/crypto only for SCRAM auth, ssh/* is not imported (according to govulncheck)
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/text v0.24.0 // indirect
)
