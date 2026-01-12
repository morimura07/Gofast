# Quick Start Guide

## Automated Setup (Recommended)

Run the PowerShell setup script:

```powershell
.\setup.ps1
```

This will:
- ✅ Check prerequisites (Docker, Docker Compose)
- ✅ Generate JWT keys (if missing)
- ✅ Start PostgreSQL
- ✅ Run database migrations
- ✅ Seed development user
- ✅ Start all services (core + client)

### Script Options

```powershell
# Skip key generation (if keys already exist)
.\setup.ps1 -SkipKeys

# Skip migrations (if already run)
.\setup.ps1 -SkipMigrations

# Skip user seeding
.\setup.ps1 -SkipSeed

# Start only core services (no frontend)
.\setup.ps1 -CoreOnly

# Combine options
.\setup.ps1 -SkipKeys -SkipMigrations
```

## Manual Setup

If you prefer manual setup or the script fails:

### 1. Generate JWT Keys
```powershell
openssl genpkey -algorithm Ed25519 -out app/private.pem
openssl pkey -in app/private.pem -pubout -out app/public.pem
```

### 2. Start PostgreSQL
```powershell
docker compose up -d postgres
```

### 3. Run Migrations
```powershell
cd app/service-core/storage
goose -dir migrations postgres "host=localhost user=postgres password=postgres dbname=postgres sslmode=disable" up
cd ../../..
```

### 4. Start Services

**Core + Client:**
```powershell
docker compose -f docker-compose.yml -f docker-compose.client.yml up --build
```

**Core only:**
```powershell
docker compose up --build
```

## Access the Application

Once running:
- **Frontend UI**: http://localhost:3000
- **Backend API**: http://localhost:4000
- **OAuth Proxy**: http://localhost:8080
- **PostgreSQL**: localhost:5432

## Stop Services

```powershell
.\stop.ps1
```

Or manually:
```powershell
docker compose down
```

## Troubleshooting

### Port Already in Use
If ports 3000, 4000, 5432, or 8080 are in use:
- Stop conflicting services
- Or modify ports in `docker-compose.yml`

### JWT Keys Missing
Ensure `app/private.pem` and `app/public.pem` exist before starting.

### Database Connection Errors
Wait 10-15 seconds after starting PostgreSQL before running migrations.

### Prerequisites Not Found
- **Docker**: Install [Docker Desktop](https://www.docker.com/products/docker-desktop)
- **OpenSSL**: Usually pre-installed on Windows 10/11, or install via Git Bash
- **Goose**: `go install github.com/pressly/goose/v3/cmd/goose@latest`
- **psql**: Install PostgreSQL client tools (optional, for seeding)

## Development User

The setup script seeds a development user:
- **Email**: dev@test.com
- **ID**: 00000000-0000-0000-0000-000000000000
- **Access**: Full permissions

You can also set `DEV_USER_ID=00000000-0000-0000-0000-000000000000` in your environment to bypass authentication in dev mode.
