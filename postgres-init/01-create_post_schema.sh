#!/bin/bash
set -e

# Connect to the specified database and create a table if it does not exist
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "mydatabase4" <<-EOSQL
    CREATE TABLE IF NOT EXISTS public.posts (
        id SERIAL PRIMARY KEY,
        title VARCHAR(255),
        author VARCHAR(255)
    );
EOSQL

