# Project Structure

The project follows standard Go layout conventions (similar to `golang-standards/project-layout`), ensuring clean separation between application entry points and internal packages.

```text
rms-backend/
├── cmd/
│   ├── api/
│   │   └── main.go           # Main application entry point. Wires dependencies, configs, and starts HTTP server.
│   └── hashpw/               # CLI utility tool for generating bcrypt hashes for manual DB seeding.
├── docs/                     # Documentation files (if any API specs or extra docs exist).
├── internal/                 # Private application and library code. Cannot be imported by other projects.
│   ├── config/               # Configuration loader (reads from .env and environment variables).
│   ├── db/                   # Database connection initialization and connection pool management (MySQL).
│   ├── dto/                  # Data Transfer Objects (Request/Response structs specific to handlers).
│   ├── handlers/             # HTTP controllers. One file per module (e.g., po_handler.go, pr_handler.go).
│   ├── middleware/           # HTTP middlewares (JWT Auth, RBAC, Rate Limiting, Security Headers).
│   ├── models/               # Core domain entities reflecting the database schema.
│   ├── repository/           # Data access layer. SQL queries, transactions, and DB interactions.
│   ├── routes/               # API Router configuration. Maps URLs to handlers and attaches middleware.
│   ├── service/              # Business logic layer (Auth, Mail, Export caching logic).
│   └── utils/                # Helper functions (JWT generation, password hashing, pagination, response formatters).
├── migrations/               # Database migration scripts.
├── postman/                  # Postman collections and environments for testing the API.
├── storage/                  # Local storage directory (e.g., for cached Excel exports, uploaded files).
├── .env.example              # Template for environment variables.
├── Dockerfile                # Docker image definition for containerized deployment.
├── Jenkinsfile               # CI/CD pipeline definition for Jenkins.
├── README.md                 # Project instruction manual.
├── rms_normalized.sql        # The normalized database schema dump.
├── seed.sql                  # Initial database seeding script (super admin, initial roles, region).
├── go.mod                    # Go module dependencies.
└── go.sum                    # Go module dependency checksums.
```

### Key Directories Explained

- **`cmd/`**: Contains the main executables. `cmd/api/main.go` is where the application boots up, connects to MySQL and Redis, initializes the Chi router, and starts listening on the configured port.
- **`internal/handlers/`**: Translates HTTP requests to business operations. For example, `po_handler.go` parses the PO creation JSON, passes it to the repository, and returns a JSON response.
- **`internal/middleware/`**: Crucial for security. Contains `RequireAccess` which checks if a user has the right to access a route.
- **`internal/models/models.go`**: The single source of truth for struct definitions used across the app (Admin, PO, PR, Region, etc.).
