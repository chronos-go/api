SELECT 'CREATE DATABASE chronos_dev'
WHERE NOT EXISTS (
    SELECT FROM pg_database WHERE datname = 'chronos_dev'
)\gexec
