# GoFast Project Setup Script
# This script automates the setup and startup of the GoFast project

param(
    [switch]$SkipKeys,
    [switch]$SkipMigrations,
    [switch]$SkipSeed,
    [switch]$CoreOnly
)

$ErrorActionPreference = 'Stop'
$ProjectRoot = $PSScriptRoot
$AppDir = Join-Path $ProjectRoot 'app'

# Colors for output
function Write-Step {
    param([string]$Message)
    Write-Host ([Environment]::NewLine + '▶ ' + $Message) -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host ('✓ ' + $Message) -ForegroundColor Green
}

function Write-Error {
    param([string]$Message)
    Write-Host ('✗ ' + $Message) -ForegroundColor Red
}

function Write-Info {
    param([string]$Message)
    Write-Host ('  ' + $Message) -ForegroundColor Gray
}

# Check prerequisites
Write-Step 'Checking prerequisites...'

# Check Docker
try {
    $dockerVersion = docker --version 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw 'Docker not found'
    }
    Write-Success ('Docker found: ' + $dockerVersion)
} catch {
    Write-Error 'Docker is not installed or not running. Please install Docker Desktop.'
    exit 1
}

# Check Docker Compose
try {
    $composeVersion = docker compose version 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw 'Docker Compose not found'
    }
    Write-Success ('Docker Compose found: ' + $composeVersion)
} catch {
    Write-Error 'Docker Compose is not available. Please ensure Docker Desktop is installed.'
    exit 1
}

# Check if Docker is running
try {
    docker ps | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw 'Docker daemon not running'
    }
    Write-Success 'Docker daemon is running'
} catch {
    Write-Error 'Docker daemon is not running. Please start Docker Desktop.'
    exit 1
}

# Check OpenSSL (for key generation)
$opensslAvailable = $false
try {
    $opensslVersion = openssl version 2>&1
    if ($LASTEXITCODE -eq 0) {
        $opensslAvailable = $true
        Write-Success ('OpenSSL found: ' + $opensslVersion)
    }
} catch {
    Write-Info 'OpenSSL not found in PATH (keys may need to be generated manually)'
}

# Step 1: Generate JWT keys
if (-not $SkipKeys) {
    Write-Step 'Generating JWT keys...'
    
    $privateKeyPath = Join-Path $AppDir 'private.pem'
    $publicKeyPath = Join-Path $AppDir 'public.pem'
    
    if ((Test-Path $privateKeyPath) -and (Test-Path $publicKeyPath)) {
        Write-Info 'JWT keys already exist, skipping generation'
    } else {
        if ($opensslAvailable) {
            Write-Info 'Generating private key...'
            openssl genpkey -algorithm Ed25519 -out $privateKeyPath
            if ($LASTEXITCODE -ne 0) {
                Write-Error 'Failed to generate private key'
                exit 1
            }
            
            Write-Info 'Generating public key...'
            openssl pkey -in $privateKeyPath -pubout -out $publicKeyPath
            if ($LASTEXITCODE -ne 0) {
                Write-Error 'Failed to generate public key'
                exit 1
            }
            
            Write-Success 'JWT keys generated successfully'
        } else {
            Write-Error 'OpenSSL not available. Please generate keys manually:'
            Write-Info '  openssl genpkey -algorithm Ed25519 -out app/private.pem'
            Write-Info '  openssl pkey -in app/private.pem -pubout -out app/public.pem'
            exit 1
        }
    }
} else {
    Write-Info 'Skipping JWT key generation'
}

# Step 2: Start PostgreSQL
Write-Step 'Starting PostgreSQL database...'
docker compose -f docker-compose.yml up -d postgres
if ($LASTEXITCODE -ne 0) {
    Write-Error 'Failed to start PostgreSQL'
    exit 1
}

Write-Info 'Waiting for PostgreSQL to be ready...'
$maxAttempts = 30
$attempt = 0
$postgresReady = $false

while ($attempt -lt $maxAttempts) {
    Start-Sleep -Seconds 2
    $attempt++
    
    try {
        $result = docker exec anerd-fast2-postgres pg_isready -U postgres 2>&1
        if ($LASTEXITCODE -eq 0) {
            $postgresReady = $true
            break
        }
    } catch {
        # Continue waiting
    }
    
    Write-Info ('  Attempt ' + $attempt + '/' + $maxAttempts + '...')
}

if (-not $postgresReady) {
    Write-Error ('PostgreSQL failed to become ready after ' + $maxAttempts + ' attempts')
    exit 1
}

Write-Success 'PostgreSQL is ready'

# Step 3: Run database migrations
if (-not $SkipMigrations) {
    Write-Step 'Running database migrations...'
    
    # Check if goose is available
    $gooseAvailable = $false
    try {
        $gooseVersion = goose -version 2>&1
        if ($LASTEXITCODE -eq 0) {
            $gooseAvailable = $true
        }
    } catch {
        # Goose not found
    }
    
    if ($gooseAvailable) {
        $storageDir = Join-Path $AppDir 'service-core\storage'
        Push-Location $storageDir
        
        try {
            goose -dir migrations postgres 'host=localhost user=postgres password=postgres dbname=postgres sslmode=disable' up
            if ($LASTEXITCODE -ne 0) {
                Write-Error 'Failed to run migrations'
                exit 1
            }
            Write-Success 'Database migrations completed'
        } finally {
            Pop-Location
        }
    } else {
        Write-Info 'Goose not found. Skipping migrations.'
        Write-Info 'Install goose: go install github.com/pressly/goose/v3/cmd/goose@latest'
        Write-Info 'Then run: cd app/service-core/storage; goose -dir migrations postgres ''host=localhost user=postgres password=postgres dbname=postgres sslmode=disable'' up'
    }
} else {
    Write-Info 'Skipping database migrations'
}

# Step 4: Seed development user (optional)
if (-not $SkipSeed) {
    Write-Step 'Seeding development user...'
    
    # Check if psql is available
    $psqlAvailable = $false
    try {
        $psqlVersion = psql --version 2>&1
        if ($LASTEXITCODE -eq 0) {
            $psqlAvailable = $true
        }
    } catch {
        # psql not found
    }
    
    if ($psqlAvailable) {
        $env:PGPASSWORD = 'postgres'
        $sql = 'INSERT INTO users (id, email, access, sub) VALUES (''00000000-0000-0000-0000-000000000000'', ''dev@test.com'', 16380, ''dev'') ON CONFLICT (id) DO NOTHING;'
        
        try {
            $result = psql -h localhost -p 5432 -U postgres -d postgres -c $sql 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Success 'Development user seeded (or already exists)'
            } else {
                Write-Info ('Could not seed user (may need migrations first): ' + $result)
            }
        } catch {
            $errorMsg = $_.Exception.Message
            Write-Info ('Could not seed user: ' + $errorMsg)
        }
    } else {
        Write-Info 'psql not found. Skipping user seeding.'
        Write-Info 'You can seed manually using the seed_dev_user.sh script or psql'
    }
} else {
    Write-Info 'Skipping development user seeding'
}

# Step 5: Start services
Write-Step 'Starting services...'

if ($CoreOnly) {
    Write-Info 'Starting core services only (backend + database)...'
    docker compose -f docker-compose.yml up --build
} else {
    Write-Info 'Starting all services (core + client)...'
    docker compose -f docker-compose.yml -f docker-compose.client.yml up --build
}

if ($LASTEXITCODE -ne 0) {
    Write-Error 'Failed to start services'
    exit 1
}

Write-Host ""
Write-Success 'Setup complete! Services are starting...'
Write-Host ""
Write-Host 'Access the application at:' -ForegroundColor Yellow
Write-Host '  - Frontend UI: http://localhost:3000' -ForegroundColor White
Write-Host '  - Backend API: http://localhost:4000' -ForegroundColor White
Write-Host '  - OAuth Proxy: http://localhost:8080' -ForegroundColor White
Write-Host ""
$msg = 'To stop services, press Ctrl+C or run: docker compose down'
Write-Host $msg -ForegroundColor Gray
