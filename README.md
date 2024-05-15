This is the back-end of asan, written in Go

## Getting Started

This project utilizes [golang-migrate](https://pkg.go.dev/github.com/golang-migrate/migrate/v4) for managing database migrations. You can streamline the migration process by incorporating a Makefile. Ensure that golang-migrate is installed according to the documentation.

There are many ways to use golang-migrate, one of them is to use it as a [CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate#with-go-toolchain), for example:
```bash
go install -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

To create a new migration:
```bash
migrate create -ext sql -dir ./postgres_migrations -seq migration_name
```

Here's an example command to apply all up migrations:
```bash
migrate -path migrations -database pgx5://user:password@localhost:5432/asan?sslmode=disable -verbose up
```

To run the server:

```bash
go run server.go
```

Open [http://localhost:8080](http://localhost:8080) for GraphQL Playground (Graphiql).

## Learn More

This project uses these following libraries

- [gqlgen](https://pkg.go.dev/github.com/wabain/gqlgen) - a library for quickly creating strictly typed graphql servers in golang.
- [pgx](https://pkg.go.dev/github.com/jackc/pgx/v5) - a pure Go driver and toolkit for PostgreSQL.
- [mongo](https://www.mongodb.com/docs/drivers/go/current/) - mongodb driver.
