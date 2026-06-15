# Database Migrations with golang-migrate

This project uses **golang-migrate** for versioning and applying database migrations.

---

## 1. CLI Usage (`migrate` command)

The `migrate` CLI is used during development to generate and apply migrations.

### A. Installation
If not already installed, you can install the CLI via Homebrew (macOS):
```bash
brew install golang-migrate
```

### B. Creating New Migrations
To generate new migration files sequentially:
```bash
migrate create -ext sql -dir db/migrations -seq <migration_name>
```
Example:
```bash
migrate create -ext sql -dir db/migrations -seq add_index_to_objects
```
This generates:
* `db/migrations/000002_add_index_to_objects.up.sql`
* `db/migrations/000002_add_index_to_objects.down.sql`

### C. Running Migrations

To run migrations, you need to provide a database connection URL. For MySQL:
`mysql://<username>:<password>@tcp(<host>:<port>)/<database_name>`

#### Run All Pending Migrations (Up)
```bash
migrate -path db/migrations -database "mysql://root:secret@tcp(127.0.0.1:3306)/cistern" up
```

#### Rollback the Last Migration (Down 1 step)
```bash
migrate -path db/migrations -database "mysql://root:secret@tcp(127.0.0.1:3306)/cistern" down 1
```

#### Force a Specific Version (e.g., if database is in dirty state)
```bash
migrate -path db/migrations -database "mysql://root:secret@tcp(127.0.0.1:3306)/cistern" force 1
```

