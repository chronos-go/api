env "local" {
  src = "file://db/schema"
  url = "postgres://postgres:postgres@localhost:5433/chronos?sslmode=disable"
  dev = "postgres://postgres:postgres@localhost:5433/chronos_dev?sslmode=disable"

  migration {
    dir = "file://db/migrations"
  }
}
