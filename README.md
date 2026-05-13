# Chronos API

API desenvolvida para a disciplina WEB II (DIM0547).

## Sprint 2 - PostgreSQL local

A Sprint 2 usa PostgreSQL como banco de dados real. Para desenvolvimento local, o projeto possui um `docker-compose.yml` com um container PostgreSQL.

### Requisitos

- Docker
- Docker Compose
- Go

### Subir o banco

```powershell
docker compose up -d
```

Esse comando sobe um PostgreSQL local com:

```text
database: chronos
user: postgres
password: postgres
port: 5432
```

### String de conexao

Use a variavel abaixo para conectar a API ao banco:

```text
DATABASE_URL=postgres://postgres:postgres@localhost:5432/chronos?sslmode=disable
```

Existe um arquivo `.env.example` com esse valor de referencia.

### Verificar se o banco esta rodando

```powershell
docker compose ps
```

### Parar o banco

```powershell
docker compose down
```

### Remover os dados locais do banco

Use apenas se quiser apagar o volume local do PostgreSQL:

```powershell
docker compose down -v
```

### Rodar a API

Com o banco rodando, execute:

```powershell
go run .
```

Por padrao, a API sobe em:

```text
http://localhost:8080
```

### Rodar os testes

```powershell
go test ./...
```
