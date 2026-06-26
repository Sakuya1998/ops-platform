SELECT 'CREATE DATABASE auth_svc'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'auth_svc')\gexec

SELECT 'CREATE DATABASE iam_svc'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'iam_svc')\gexec

SELECT 'CREATE DATABASE audit_svc'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'audit_svc')\gexec

SELECT 'CREATE DATABASE notify_svc'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'notify_svc')\gexec
