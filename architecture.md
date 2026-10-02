# System Architecture

## 1. High-Level Architecture

The RMS Backend is designed as a monolithic REST API service written in Go. It acts as the intermediary between the React/Vite frontend and the persistent storage layers (MySQL and Redis). 

The architecture follows a standard layered (N-Tier) approach commonly used in Go web services to ensure separation of concerns.

## 2. Technology Stack
- **Language:** Go 1.22
- **Router:** `go-chi/chi` (lightweight, idiomatic routing)
- **Database:** MySQL 8+ (accessed via `database/sql` and `go-sql-driver/mysql`)
- **Cache & Token Store:** Redis (accessed via `redis/go-redis/v9`)
- **Authentication:** JWT (JSON Web Tokens) using HS256 algorithm.
- **Excel Generation:** `xuri/excelize/v2`

## 3. Layered Application Structure

The application flow typically follows: `Router -> Middleware -> Handler -> Service -> Repository -> Database`.

### 3.1 Handlers (`internal/handlers`)
Responsible for parsing incoming HTTP requests (JSON body, query params, URL segments), calling the appropriate Service layer methods, and formatting the JSON HTTP response. 
- Avoids business logic.
- Maps domain errors to appropriate HTTP status codes (400, 401, 403, 404, 500).

### 3.2 Services (`internal/service`)
Contains the core business logic. 
- E.g., `AuthService` handles hashing passwords, generating JWTs, storing tokens in Redis.
- E.g., `MailService` handles sending activation emails.

### 3.3 Repositories (`internal/repository`)
Responsible for all database interactions.
- Executes SQL queries using parameterized inputs to prevent SQL Injection.
- Handles database transactions (`sql.Tx`) to ensure atomicity across multiple table inserts/updates (e.g., saving a PO and its items together).

### 3.4 Models (`internal/models`)
Defines the struct representations of the database tables and JSON payloads.
- Acts as the Data Transfer Objects (DTOs) between layers.

## 4. Security Architecture

- **Stateless Auth with Stateful Revocation:** JWTs are stateless, but refresh tokens are tracked in Redis. Upon logout or password change, the current access token is added to a Redis blacklist to simulate immediate revocation, and refresh tokens are deleted.
- **RBAC Middleware:** The `RequireAccess(slug)` middleware intercepts requests and verifies if the authenticated user's Role possesses the required permission slug by querying the database/cache.
- **Rate Limiting:** Redis-backed rate limiters (`middleware/rate_limit.go`, `login_throttle.go`) protect endpoints from brute-force attacks and abuse.
- **CORS & Headers:** Explicit CORS configuration and security headers (`X-Content-Type-Options`, `X-Frame-Options`) are injected via middleware.

## 5. Caching Strategy
- **Export Caching:** The Excel export feature (`GET /api/po/export`) uses a file-based caching mechanism. The cache key is a hash of the query parameters. If a cache hit occurs within the TTL (e.g., 20 mins), the file is served directly from disk, bypassing the database entirely.
