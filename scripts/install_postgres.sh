#!/bin/bash

apt-get update
apt-get install -y --no-install-recommends openssh-server
echo "shared_preload_libraries = 'pg_stat_statements'" >> /var/lib/postgresql/data/postgresql.conf
psql -U postgres -d stat-db -c 'CREATE EXTENSION pg_stat_statements;'

