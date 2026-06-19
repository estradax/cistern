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

- **Create Bucket**:
  ```bash
  ./cistern buckets create '{"bucket_key": "my-bucket", "owner_id": "client-uuid-here"}'
  ```
- **Read Bucket** (supports raw ID or JSON):
  ```bash
  ./cistern buckets read 'bucket-uuid-here'
  # OR
  ./cistern buckets read '{"id": "bucket-uuid-here"}'
  ```
- **Edit / Update Bucket**:
  ```bash
  ./cistern buckets update '{"id": "bucket-uuid-here", "bucket_key": "updated-bucket-key", "owner_id": "client-uuid-here"}'
  # OR
  ./cistern buckets edit '{"id": "bucket-uuid-here", "bucket_key": "updated-bucket-key", "owner_id": "client-uuid-here"}'
  ```
- **Delete Bucket** (supports raw ID or JSON):
  ```bash
  ./cistern buckets delete 'bucket-uuid-here'
  ```

- **Upload Object**:
  ```bash
  ./cistern objects upload '{"bucket_id": "bucket-uuid-here", "object_key": "my/path.txt"}' /path/to/local/file
  ```
- **Read Object Metadata** (supports raw ID or JSON):
  ```bash
  ./cistern objects read 'object-uuid-here'
  # OR
  ./cistern objects read '{"id": "object-uuid-here"}'
  ```
- **Download Object**:
  ```bash
  ./cistern objects download 'object-uuid-here' /path/to/destination
  ```
- **Delete Object** (supports raw ID or JSON):
  ```bash
  ./cistern objects delete 'object-uuid-here'
  ```
- **List Objects in Bucket** (supports raw ID or JSON):
  ```bash
  ./cistern objects list 'bucket-uuid-here'
  ```
- **Generate Presigned URL**:
  ```bash
  ./cistern objects presign '{"object_key": "my/path.txt", "method": "GET", "expires_in": 3600}'
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

---

## 5. Go Library Usage (`internal/bucket`)

To interact with buckets programmatically in the Go code:

```go
import (
	"context"
	"github.com/estradax/cistern/internal/bucket"
	"github.com/jmoiron/sqlx"
)

// Initialize the Repository with a database pool
repo := bucket.NewRepository(db)

// Create a bucket
b, err := repo.Create(ctx, bucket.CreateBucketInput{BucketKey: "my-bucket", OwnerID: "client-uuid"})

// Get a bucket by ID
b, err := repo.Get(ctx, "bucket-uuid")
```

---

## 6. Go Library Usage (`internal/object` & `internal/storage`)

To interact with objects and storage programmatically in the Go code:

```go
import (
	"context"
	"bytes"
	"github.com/estradax/cistern/internal/object"
	"github.com/estradax/cistern/internal/storage"
	"github.com/jmoiron/sqlx"
)

// Initialize the storage driver (e.g. Local Driver)
store, err := storage.NewLocalDriver("./data/storage")

// Initialize the Repository with a database pool
repo := object.NewRepository(db)

// Initialize the Service
svc := object.NewService(repo, store)

// Upload an object
obj, err := svc.Upload(ctx, "bucket-uuid", "documents/notes.txt", "text/plain", bytes.NewReader([]byte("my content")))

// Download an object payload (caller must close the reader)
meta, reader, err := svc.Download(ctx, "object-uuid")
defer reader.Close()

// Delete an object (logical metadata delete and physical payload cleanup)
err = svc.Delete(ctx, "object-uuid")
```

---

## 7. Running Tests

### A. Running Tests CLI
To run the full suite of unit and integration tests, run:
```bash
make test
```
*(This is a convenience command that executes `go test -v ./...` under the hood).*

To run tests for a specific package only:
```bash
go test -v ./internal/client
```

### B. Requirements & Environment
* **MySQL Server**: A running MySQL server must be active and accessible at the host/port specified in `.env.testing`.
* **Testing Configuration (`.env.testing`)**: Holds the baseline connection DSN and storage folder details. Example:
  ```env
  DB_DSN=root:12345678@tcp(127.0.0.1:3306)/cistern_test?parseTime=true&multiStatements=true
  STORAGE_DIR=./data/storage
  ```

### C. Testing Harness & Isolation Strategy (`internal/testutil`)
The database-backed tests leverage a helper package located at [internal/testutil/testutil.go](file:///Users/hansel/project/cistern/internal/testutil/testutil.go) to ensure test isolation:

1. **Dynamic Database Isolation**:
   To support package-level isolation (preventing parallel execution conflicts), [SetupTestDB](file:///Users/hansel/project/cistern/internal/testutil/testutil.go#L34) reads `DB_DSN` from `.env.testing` and appends the package name of the test being run to the database name (e.g., `cistern_test_client`, `cistern_test_apikey`).
2. **Dynamic Database Schema Setup**:
   For each test suite run, the harness dynamically drops the package-specific test database if it already exists, recreates it, and runs all schema migrations located in the `db/migrations/` directory.
3. **Database Cleanup**:
   Use [CleanDatabase](file:///Users/hansel/project/cistern/internal/testutil/testutil.go#L113) to truncate tables between test runs (disabling and re-enabling foreign key checks) to maintain an isolated database state.

#### Usage Example in Tests:
```go
import "github.com/estradax/cistern/internal/testutil"

func TestMyRepository(t *testing.T) {
	// Setup isolated test database and apply migrations
	db := testutil.SetupTestDB(t)
	defer db.Close()

	t.Run("my subtest", func(t *testing.T) {
		// Clean tables before each subtest to ensure clean state
		testutil.CleanDatabase(t, db)
		
		// Execute test assertions here...
	})
}
```

