#!/bin/bash

docker compose up --no-start
docker compose up -d postgres
docker exec -i stat-bot-postgres bash < ./scripts/install_postgres.sh
docker compose restart postgres
docker exec -i stat-bot-postgres bash < ./scripts/ssh_conf.sh
docker compose up -d statserver