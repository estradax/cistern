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

---

## 2. CLI Usage (`cistern` command)

The `cistern` CLI tool is used to perform CRUD operations on clients.

### A. Build the CLI
```bash
go build -o cistern ./cmd/cistern
```

### B. CLI Commands
- **Create**:
  ```bash
  ./cistern clients create '{"name": "Client Name"}'
  ```
- **Read** (supports raw ID or JSON):
  ```bash
  ./cistern clients read 'client-uuid-here'
  # OR
  ./cistern clients read '{"id": "client-uuid-here"}'
  ```
- **Update**:
  ```bash
  ./cistern clients update '{"id": "client-uuid-here", "name": "Updated Name"}'
  ```
- **Delete** (supports raw ID or JSON):
  ```bash
  ./cistern clients delete 'client-uuid-here'
  ```

- **Generate API Key**:
  ```bash
  ./cistern apikeys generate '{"client_id": "client-uuid-here", "name": "Key Name"}'
  ```
- **Read API Key** (supports raw ID or JSON):
  ```bash
  ./cistern apikeys read 'apikey-uuid-here'
  # OR
  ./cistern apikeys read '{"id": "apikey-uuid-here"}'
  ```
- **Delete API Key** (supports raw ID or JSON):
  ```bash
  ./cistern apikeys delete 'apikey-uuid-here'
  ```

---

## 3. Go Library Usage (`internal/client`)

To interact with clients programmatically in the Go code:

```go
import (
	"context"
	"github.com/estradax/cistern/internal/client"
	"github.com/jmoiron/sqlx"
)

// Initialize the Repository with a database pool
repo := client.NewRepository(db)

// Create a client
c, err := repo.Create(ctx, client.CreateClientInput{Name: "Acme Corp"})

// Get a client by ID
c, err := repo.Get(ctx, "client-uuid")
```

---

## 4. Go Library Usage (`internal/apikey`)

To interact with API keys programmatically in the Go code:

```go
import (
	"context"
	"github.com/estradax/cistern/internal/apikey"
	"github.com/jmoiron/sqlx"
)

// Initialize the Repository with a database pool
repo := apikey.NewRepository(db)

// Generate an API key
res, err := repo.Create(ctx, apikey.CreateAPIKeyInput{ClientID: "client-uuid", Name: &keyName})

// Get an API key by ID
key, err := repo.Get(ctx, "apikey-uuid")
```
