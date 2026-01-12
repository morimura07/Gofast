#!/bin/bash

echo "Seeding development user into the database..."

# This script inserts a development user into the database for testing purposes.

# Database connection details (adjust if your docker-compose.yml is different)
DB_HOST="localhost"
DB_PORT="5432"
DB_USER="postgres"
DB_PASSWORD="postgres"
DB_NAME="postgres"

# Development user details
DEV_USER_ID="00000000-0000-0000-0000-000000000000"
DEV_USER_EMAIL="dev@test.com"
# Using auth.UserAccess value (see app/pkg/auth/auth.go)
DEV_USER_ACCESS=16380

# Construct the psql command
PSQL_COMMAND="psql --host=$DB_HOST --port=$DB_PORT --username=$DB_USER --dbname=$DB_NAME"

# SQL statement to insert the development user
# It uses ON CONFLICT DO NOTHING to avoid errors if the user already exists.
SQL="INSERT INTO users (id, email, access, sub) VALUES ('$DEV_USER_ID', '$DEV_USER_EMAIL', $DEV_USER_ACCESS, 'dev') ON CONFLICT (id) DO NOTHING;"

# Execute the SQL command
PGPASSWORD=$DB_PASSWORD $PSQL_COMMAND -c "$SQL"

echo "Development user with ID $DEV_USER_ID seeded successfully (if it didn't already exist)."

