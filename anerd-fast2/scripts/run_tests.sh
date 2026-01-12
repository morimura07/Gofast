#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Get the project root directory (one level up from scripts/)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Track timing
START_TIME=$(date +%s)
STRIPE_PID=""

print_header() {
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC} ${BOLD}$1${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
}

print_step() {
    echo -e "${CYAN}▶${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_elapsed() {
    local end_time=$(date +%s)
    local elapsed=$((end_time - START_TIME))
    local minutes=$((elapsed / 60))
    local seconds=$((elapsed % 60))
    echo -e "${CYAN}⏱${NC}  Elapsed time: ${minutes}m ${seconds}s"
}

cleanup() {
    print_header "CLEANUP"

    # Stop Stripe CLI if running
    if [ -n "$STRIPE_PID" ]; then
        print_step "Stopping Stripe CLI..."
        kill $STRIPE_PID 2>/dev/null || true
        print_success "Stripe CLI stopped"
    fi

    print_step "Stopping docker services..."
    cd "$PROJECT_ROOT"
    docker compose -f docker-compose.yml -f docker-compose.client.yml -f docker-compose.test.yml stop 2>/dev/null || true
    print_success "Services stopped"
    print_elapsed
}

# Set trap to cleanup on exit (success or failure)
trap cleanup EXIT

print_header "GOFAST TEST SUITE"
echo -e "Started at: $(date)"
echo -e "Project root: $PROJECT_ROOT"

# ============================================================================
# PREREQUISITES
# ============================================================================
print_header "CHECKING PREREQUISITES"

if [ ! -f "$PROJECT_ROOT/app/private.pem" ] || [ ! -f "$PROJECT_ROOT/app/public.pem" ]; then
    print_step "Generating Ed25519 keys..."
    cd "$PROJECT_ROOT"
    make keys
    print_success "Keys generated"
else
    print_success "Keys already exist"
fi

# ============================================================================
# PREPARE
# ============================================================================
print_header "PREPARING CODE"

cd "$PROJECT_ROOT"

print_step "Generating protobuf files..."
make gen
print_success "Protobuf generation completed"

print_step "Generating sqlc code..."
make sql
print_success "SQLC generation completed"

print_step "Formatting code..."
make format
print_success "Code formatting completed"

# ============================================================================
# DOCKER SERVICES
# ============================================================================
print_header "CLEANING UP DOCKER SERVICES"
make down

print_header "STARTING DOCKER SERVICES"

print_step "Building and starting services..."
cd "$PROJECT_ROOT"


DEV_USER_ID=00000000-0000-0000-0000-000000000000 \
POSTGRES_HOST=postgres \
POSTGRES_PORT=5432 \
POSTGRES_DB=postgres \
POSTGRES_USER=postgres \
POSTGRES_PASSWORD=postgres \
STRIPE_API_KEY="${STRIPE_API_KEY:-}" \
STRIPE_WEBHOOK_SECRET="${STRIPE_WEBHOOK_SECRET:-}" \
STRIPE_PRICE_ID_BASIC="${STRIPE_PRICE_ID_BASIC:-}" \
STRIPE_PRICE_ID_PRO="${STRIPE_PRICE_ID_PRO:-}" \
EMAIL_FROM="${EMAIL_FROM:-}" \
POSTMARK_API_KEY="${POSTMARK_API_KEY:-}" \
R2_ACCESS_KEY="${R2_ACCESS_KEY:-}" \
R2_SECRET_KEY="${R2_SECRET_KEY:-}" \
R2_ENDPOINT="${R2_ENDPOINT:-}" \
BUCKET_NAME="${BUCKET_NAME:-}" \
docker compose -f docker-compose.yml -f docker-compose.client.yml -f docker-compose.test.yml up -d --build --wait
print_success "Docker services started and healthy"

# Start Stripe CLI if available and API key is set
if command -v stripe &> /dev/null && [ -n "$STRIPE_API_KEY" ]; then
    print_step "Starting Stripe webhook listener..."
    make -C "$PROJECT_ROOT" stripe &
    STRIPE_PID=$!
    sleep 2
    print_success "Stripe CLI listening (PID: $STRIPE_PID)"
else
    print_warning "Stripe CLI not available or STRIPE_API_KEY not set - subscription tests will be skipped"
fi

# ============================================================================
# DATABASE SETUP
# ============================================================================
print_header "DATABASE SETUP"

print_step "Running migrations..."
cd "$PROJECT_ROOT"
make migrate
print_success "Migrations completed"

print_step "Seeding dev user..."
cd "$SCRIPT_DIR"
sh ./seed_dev_user.sh
print_success "Dev user seeded"

# ============================================================================
# BACKEND TESTS (Go)
# ============================================================================
print_header "BACKEND TESTS (Go)"

cd "$PROJECT_ROOT/app"

print_step "Building Go packages..."
go build ./...
print_success "Go build completed"

print_step "Linting Go code..."
golangci-lint run
print_success "Go lint passed"

print_step "Running Go tests with coverage and race detection..."
go test -cover -race ./...
print_success "Go tests passed"

cd "$PROJECT_ROOT/app/service-core"

print_step "Building Go packages in service-core..."
go build ./...
print_success "Go build in service-core completed"

print_step "Linting Go code in service-core..."
golangci-lint run
print_success "Go lint in service-core passed"

print_step "Running Go tests with coverage and race detection in service-core..."
go test -cover -race ./...
print_success "Go tests in service-core passed"

# ============================================================================
# FRONTEND LINT (SvelteKit)
# ============================================================================
print_header "FRONTEND LINT (SvelteKit)"

cd "$PROJECT_ROOT/app/service-client"

print_step "Installing npm dependencies..."
npm ci
print_success "Dependencies installed"

print_step "Running type check (svelte-check)..."
npm run check
print_success "Type check passed"

print_step "Running linter..."
npm run lint
print_success "Lint check passed"

# ============================================================================
# E2E TESTS (Playwright)
# ============================================================================
print_header "E2E TESTS (Playwright)"

cd "$PROJECT_ROOT/e2e"

print_step "Installing npm dependencies..."
npm ci
print_success "Dependencies installed"

print_step "Running Playwright tests..."
npm run test
print_success "E2E tests passed"

# ============================================================================
# SUMMARY
# ============================================================================
print_header "TEST SUITE COMPLETED SUCCESSFULLY"
echo -e "${GREEN}${BOLD}All tests passed!${NC}"
print_elapsed
echo ""
