### Cache Debouncing

When multiple clients fetch data from Redis and there is not key in Redis,
then all the clients fetch data from db and inserts the data back to Redis.
This overwhelms the database.

So instead of all clients fetching data from Redis, let one client fetch the
missing data from Redis while other clients waits for data to be inserted
back to Redis from database.

The clients waits for specified timeout for the one client to get data from
the database. After the specified timout and there is still no data, then
all clients returns error.

## How to run

Give permission to postgres-init to execute script to create schema and users

```shell
chmod +x ./postgres-init/*.sh
```

start the docker to run the postgres databases and redis cache

```shell
docker-compose up --build
```

start the server in another terminal

```shell
go run simulate_cache_debouncing.go
```

Make sure you only see 1 client fetching data from database using
the below log line.This client ID will be random for you but there should
be be one of this line.

```shell
2024/04/15 22:33:41 fetching post for client ID: 10
```
