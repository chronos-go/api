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

Copie também as demais variáveis de `.env.example`. `JWT_SECRET` é obrigatório e
deve conter pelo menos 32 caracteres; a API não inicia com segredo padrão.

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

## Autenticação e segurança

- `POST /auth/login`: retorna access token JWT e refresh token opaco.
- `POST /auth/refresh`: rotaciona o refresh token; reutilização revoga a família.
- `POST /auth/logout`: revoga a sessão.
- `/providers` e `/services` exigem `Authorization: Bearer <access_token>`.
- Escrita em services exige role `provider`; update/delete também validam ownership.
- Login, refresh e logout possuem rate limiting configurável.
- O servidor limita o body, rejeita campos JSON desconhecidos, aplica headers de
  segurança, CORS explícito e timeouts HTTP.

O refresh token é persistido apenas como SHA-256 em `auth_sessions`; o valor em
texto puro só é entregue ao cliente no login/refresh.

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
