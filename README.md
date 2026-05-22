# Chronos API

API da disciplina WEB II (DIM0547).

## Requisitos

- Go
- Docker
- Docker Compose
- Atlas CLI

## Banco local

Subir PostgreSQL:

```powershell
docker compose up -d
```

Verificar status:

```powershell
docker compose ps
```

Dados do banco:

```text
host: localhost
port: 5433
user: postgres
password: postgres
database: chronos
database auxiliar do Atlas: chronos_dev
```

String de conexao:

```text
DATABASE_URL=postgres://postgres:postgres@localhost:5433/chronos?sslmode=disable
```

## Migrations

Arquivos principais:

```text
db/schema/001_init.sql
db/migrations/20260513000100_init.sql
atlas.hcl
```

Aplicar migrations:

```powershell
atlas migrate apply --env local
```

Ver status:

```powershell
atlas migrate status --env local
```

Validar migrations:

```powershell
atlas migrate validate --env local
```

Atualizar hash das migrations:

```powershell
atlas migrate hash --env local
```

Gerar nova migration depois de alterar `db/schema/`:

```powershell
atlas migrate diff nome_da_migration --env local
```

## Schema

O schema cria:

- `clients`
- `providers`
- `services`

Relacionamento 1:N da Sprint 2:

```text
providers -> services
```

## Rodar API

```powershell
go run .
```

URL local:

```text
http://localhost:8080
```

## Testes

```powershell
go test ./...
```

## Resetar banco local

Apaga o container e o volume local:

```powershell
docker compose down -v
```

Depois suba e aplique as migrations novamente:

```powershell
docker compose up -d
atlas migrate apply --env local
```
