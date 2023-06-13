module github.com/kiwiirc/plugin-fileuploader

go 1.16

require (
	github.com/BurntSushi/toml v1.3.2
	github.com/IGLOU-EU/go-wildcard v1.0.3
	github.com/bmizerany/pat v0.0.0-20210406213842-e4b6760bdd6f // indirect
	github.com/c2h5oh/datasize v0.0.0-20220606134207-859f65c6625b
	github.com/gin-gonic/gin v1.9.1
	github.com/go-playground/validator/v10 v10.14.1 // indirect
	github.com/go-sql-driver/mysql v1.7.1
	github.com/golang-jwt/jwt/v4 v4.5.0
	github.com/jmoiron/sqlx v1.3.5
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/mattn/go-colorable v0.1.13
	github.com/mattn/go-sqlite3 v1.14.17
	github.com/rs/zerolog v1.29.1
	github.com/rubenv/sql-migrate v1.4.0
	github.com/tus/tusd v1.11.0
	github.com/ugorji/go v1.2.7 // indirect
)

replace github.com/tus/tusd => github.com/ItsOnlyBinary/tusd v0.0.0-20230612164107-188c9c0f7acf
