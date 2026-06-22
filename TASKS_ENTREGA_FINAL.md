# Entrega Final — auditoria e backlog por funcionalidade

Prazo: **29/06/2026 às 23:59**  
Equipe: **4 pessoas**  
Estado analisado: branch `main`, commit `7bef7cd`, em 22/06/2026.

## Diagnóstico

| Critério | Estado | Lacuna principal |
|---|---|---|
| Login com JWT | Parcial | Access token existe, mas o cadastro de client não persiste dados e o fluxo de client não fecha. |
| Rotas protegidas | Não atendido | Não há middleware de autenticação/autorização no router. |
| Refresh com rotação | Não atendido | Não há refresh token, sessão, rotação, revogação ou logout. |
| Duas correções OWASP | Não comprovado | Há bcrypt e validações pontuais, mas faltam controles explícitos e demonstráveis. |
| Dez testes no CI | Quantidade atendida | Existem 48 testes e `go test ./...` passa localmente; o CI não testa PostgreSQL nem migrations. |
| CRUD PostgreSQL/sqlc | Parcial | `services` está persistido; `providers` não tem update/delete; `clients` usa memória. |
| Relacionamento 1:N | Atendido | `GET /providers/{id}/details` retorna provider com services. Falta teste integrado. |
| Documentação | Parcial | README básico e Postman sem login, refresh e rotas protegidas. |
| Vídeo | Pendente | Faltam roteiro, ensaio, gravação, upload e validação do acesso. |

## Regra para trabalho paralelo

Cada tarefa abaixo deve ser executável em branch própria e evitar alterações em `main.go`. A funcionalidade deve expor uma função de registro, como `RegisterRoutes(router, dependencies)`, para ser conectada somente na tarefa final `INT-01`.

- Cada tarefa inclui seus próprios testes.
- Migrations devem usar timestamps diferentes previamente reservados.
- Não editar manualmente arquivos gerados pelo sqlc.
- Se duas tarefas precisarem de um contrato comum, criar uma interface pequena no pacote consumidor em vez de alterar o outro pacote.
- Alterações globais em router, `.env.example`, README principal e workflow ficam reservadas às tarefas específicas indicadas.
- A atribuição das tarefas às quatro pessoas é feita no quadro; o backlog não vincula funcionalidade a integrante.

## Funcionalidade: sessões e refresh token

### AUTH-01 — Ciclo completo de sessão (P0, 10 h)

Escopo isolado: `internal/auth`, `internal/handler/auth`, nova migration de sessions e queries próprias.

- Validar segredo, issuer e TTLs; remover `dev-secret-change-me` da execução normal.
- Adicionar `jti` e tipo do token.
- Criar tabela de sessões/refresh tokens com token armazenado somente como hash, usuário, role, família, expiração, uso e revogação.
- Implementar `POST /auth/login`, `POST /auth/refresh` e `POST /auth/logout`.
- Rotacionar refresh a cada uso; rejeitar token anterior; em replay, revogar a família.
- Não imprimir `DATABASE_URL`, tokens ou segredos.
- Expor `auth.RegisterRoutes(...)`, sem alterar `main.go`.

Critérios de aceite:

- Login retorna access e refresh.
- Refresh válido entrega novo par e invalida o anterior.
- Replay, token expirado e logout são rejeitados.
- Credenciais inválidas sempre retornam mensagem genérica.
- Testes unitários e de repositório cobrem todos esses casos.

## Funcionalidade: clientes persistidos

### CLIENT-01 — CRUD completo de clients com PostgreSQL/sqlc (P0, 10 h)

Escopo isolado: `internal/handler/client`, novo repository de client, `db/query/clients.sql` e testes do recurso.

- Substituir o `map` global por interface e implementação PostgreSQL.
- Alinhar o DTO ao schema: `name`, `email`, `birth_date`, `password`; remover `phone`.
- Implementar create, get, list administrativa se necessária, update e delete.
- Hash da senha com bcrypt; senha nunca aparece na resposta.
- Implementar get-by-email para permitir autenticação do client.
- Normalizar email e validar formato, data, tamanho dos campos e senha mínima.
- Usar `r.Context()` nas operações de banco.
- Mapear unique violation para 409 e ausência para 404.
- Expor `client.RegisterRoutes(...)`, sem alterar `main.go`.

Critérios de aceite:

- CRUD sobrevive a restart do servidor.
- Client recém-cadastrado pode ser encontrado pelo serviço de login.
- Respostas 400, 404 e 409 são testadas.
- Testes de integração confirmam persistência real.

## Funcionalidade: providers e relacionamento 1:N

### PROVIDER-01 — Completar CRUD e consolidar endpoint aninhado (P1, 7 h)

Escopo isolado: `internal/handler/provider`, `internal/repository/provider_repo.go`, queries de provider e seus testes.

- Adicionar update e delete via sqlc.
- Manter create, list e get persistidos.
- Garantir que senha/hash nunca apareça em JSON.
- Manter `GET /providers/{id}/details` retornando `services: []` quando vazio.
- Documentar e testar a exclusão em cascata de services.
- Trocar `context.Background()` pelo context da requisição por meio dos métodos do repository.
- Expor `provider.RegisterRoutes(...)`, sem alterar `main.go`.

Critérios de aceite:

- Create/list/get/update/delete usam PostgreSQL.
- Endpoint aninhado retorna provider com zero ou vários services.
- Casos 400, 404 e 409 têm testes.

## Funcionalidade: autenticação de rotas e OWASP

### SECURITY-01 — Middleware JWT, roles e controle por objeto (P0, 9 h)

Escopo isolado: novo pacote `internal/middleware/security` e testes com handlers falsos.

- Ler `Authorization: Bearer` e validar assinatura, issuer, expiração e tipo `access`.
- Inserir subject e role no context usando tipo próprio.
- Criar middleware/helper de role.
- Criar política reutilizável de ownership para comparar `sub` com proprietário do recurso.
- Definir políticas: provider altera apenas seus services; client acessa/altera apenas o próprio cadastro.
- Retornar 401 para autenticação ausente/inválida e 403 para autorização negada.
- Não alterar handlers de client/service nesta tarefa; fornecer helpers e exemplos/testes de uso.

Critérios de aceite:

- Testes cobrem token ausente, esquema inválido, assinatura inválida, expirado e válido.
- Testes cobrem role incorreta e ownership de outro usuário.
- O pacote pode ser conectado a qualquer subrouter sem dependência de handlers concretos.

### SECURITY-02 — Rate limiting de login e refresh (P0, 5 h)

Escopo isolado: novo pacote `internal/middleware/ratelimit`.

- Implementar limite configurável por IP para endpoints sensíveis.
- Se viável, combinar IP e identidade normalizada sem armazenar senha/payload.
- Responder 429 de forma determinística.
- Só confiar em `X-Forwarded-For` quando proxy confiável estiver configurado.
- Expor middleware independente e testável com relógio injetável.

Critérios de aceite:

- Teste excede o limite e recebe 429.
- Janela expirada libera novas requisições.
- IPs distintos não compartilham indevidamente o mesmo contador.

### SECURITY-03 — Hardening HTTP e validação de entrada (P1, 5 h)

Escopo isolado: novo pacote `internal/httpx` e middleware de headers/limites.

- Limitar body com `http.MaxBytesReader`.
- Criar decoder comum que rejeita campos desconhecidos, JSON múltiplo e body inválido.
- Adicionar headers como `X-Content-Type-Options: nosniff` e `Cache-Control: no-store` nas respostas de token.
- Fornecer configuração de `http.Server` com read header, read, write e idle timeouts.
- Definir CORS explicitamente, sem wildcard quando houver credenciais.

Critérios de aceite:

- Testes cobrem body excessivo, campo desconhecido, JSON múltiplo e headers.
- Os helpers são reutilizáveis sem importar handlers de domínio.

## Funcionalidade: services e autorização de ownership

### SERVICE-01 — Aplicar identidade autenticada ao CRUD de services (P0, 6 h)

Escopo isolado: `internal/handler/service`, queries específicas e testes.

- Em create, obter `provider_id` do subject autenticado e ignorar/remover o campo do body.
- Em update/delete, consultar ownership e negar outro provider com 403.
- Manter list/get conforme a política definida para leitura.
- Preservar validações de preço, duração, UUID e provider inexistente.
- Expor `service.RegisterRoutes(...)`, sem alterar `main.go`.

Critérios de aceite:

- Provider A cria service associado a si mesmo.
- Provider A não altera nem exclui service de B.
- CRUD legítimo continua persistido via sqlc.
- Testes usam claims no context e repository falso; integração real fica em `TEST-01`.

## Funcionalidade: testes e pipeline

### TEST-01 — Suíte de integração end-to-end com PostgreSQL (P0, 9 h)

Escopo isolado: diretório próprio de testes de integração, sem alterar código de negócio.

- Criar harness que aplica migrations em banco isolado e limpa dados entre testes.
- Cobrir cadastro -> login -> rota protegida -> CRUD service -> endpoint aninhado -> refresh -> logout.
- Cobrir conflito, FK inválida, 401, ownership 403, 404, rate limit 429 e replay de refresh.
- Usar fixtures reprodutíveis e nunca depender da ordem dos testes.

Critérios de aceite:

- A suíte falha se PostgreSQL, migration ou query sqlc estiver quebrada.
- Passa repetidamente em ambiente limpo.
- Pelo menos 10 testes relevantes rodam no pipeline, além dos testes unitários existentes.

### CI-01 — Pipeline completo com PostgreSQL (P0, 5 h)

Escopo isolado: `.github/workflows/ci.yml` e scripts exclusivos de CI.

- Subir PostgreSQL 16 como service com healthcheck.
- Executar migrations e suíte de integração.
- Verificar `gofmt`, `go vet`, `go build`, `go test` e geração sqlc sem diff.
- Validar a versão Go configurada no runner.
- Executar `-race` se o tempo permitir e publicar resumo de cobertura.

Critérios de aceite:

- Workflow fica verde em PR limpo.
- Fica vermelho quando migration ou teste é deliberadamente quebrado.
- Branch `main` exige CI verde e review antes do merge.

## Funcionalidade: documentação e ambiente de demonstração

### DOC-01 — README, OpenAPI, Postman e seed (P1, 8 h)

Escopo isolado: documentação, coleção Postman, spec OpenAPI e script/fixture de seed.

- Documentar setup, variáveis, migrations, seed, testes e arquitetura.
- Documentar decisões OWASP e matriz de rotas públicas/protegidas/roles.
- Criar OpenAPI 3 com schemas, erros e Bearer auth.
- Atualizar Postman com captura automática de tokens, refresh, CRUD, aninhado e erros 401/403/404/409/429.
- Criar seed idempotente com client, dois providers e services, usando apenas credenciais de demo.

Critérios de aceite:

- Uma pessoa sem contexto executa o fluxo completo seguindo apenas README/Postman.
- OpenAPI e Postman correspondem às rotas finais.
- Ambiente limpo fica pronto para demonstração por comando documentado.

## Funcionalidade: integração da aplicação

### INT-01 — Compor router e resolver contratos (P0, 5 h, executar após os PRs funcionais)

Esta é a única tarefa intencionalmente dependente. Deve ser pequena porque cada funcionalidade já entrega seu registrador de rotas.

- Conectar repositories, serviços, middlewares e registradores no composition root.
- Separar grupos públicos, autenticados e exclusivos de provider/client.
- Aplicar rate limiting somente onde definido e hardening global.
- Centralizar leitura de configuração e shutdown gracioso.
- Resolver conflitos de migrations/sqlc e atualizar `.env.example`.
- Rodar toda a suíte e corrigir somente problemas de integração, sem reimplementar funcionalidades.

Critérios de aceite:

- Aplicação sobe em ambiente vazio e todos os fluxos funcionam via HTTP.
- Nenhuma rota de negócio protegida fica pública por engano.
- `go test ./...`, integração, vet, build e CI remoto ficam verdes.

## Funcionalidade: vídeo e submissão

### DELIVERY-01 — Roteiro, gravação e envio (P0, 7 h)

Pode começar em paralelo com roteiro e preparação de cenas; a gravação final usa a release candidate.

Roteiro de 12 minutos:

1. 0:00–1:00 — equipe, problema e proposta do Chronos.
2. 1:00–5:30 — cadastro/login, JWT, CRUD PostgreSQL, 1:N aninhado e erros.
3. 5:30–7:30 — refresh com rotação, 401/403 e duas correções OWASP.
4. 7:30–10:30 — arquitetura, schema/sqlc, middlewares, testes, OpenAPI e CI verde.
5. 10:30–12:00 — decisões, aprendizados, limitações e próximos passos.

- Dividir falas entre os quatro integrantes.
- Ensaiar com cronômetro e seed restaurável.
- Gravar em 1080p, conferir áudio/texto e não exibir segredos.
- Subir como YouTube não listado ou Drive.
- Testar o link em janela anônima.
- Criar tag/release final e submeter links do GitHub e vídeo no SIGAA.
- Uma segunda pessoa revisa a submissão antes do prazo.

Critérios de aceite:

- Vídeo próximo de 12 minutos cobre toda a rubrica.
- Repositório e vídeo abrem sem solicitar autorização ao professor.
- Links corretos são enviados antes de 29/06 às 23:59.

## Ordem sugerida para quatro frentes paralelas

As tarefas não são pré-atribuídas. A equipe escolhe quatro da primeira coluna e puxa as próximas conforme libera capacidade.

| Fase | Tarefas paralelas disponíveis |
|---|---|
| Início | `AUTH-01`, `CLIENT-01`, `PROVIDER-01`, `SECURITY-01`, `SECURITY-02`, `SECURITY-03`, `CI-01`, `DOC-01` |
| Com contratos de security prontos | `SERVICE-01`, continuação de `AUTH-01`/`CLIENT-01`, início de `TEST-01` |
| Após merges funcionais | `INT-01`, conclusão de `TEST-01`, atualização final de `DOC-01` |
| Release candidate | Gravação final de `DELIVERY-01`, correções, tag e submissão |

## Marcos

| Data interna | Resultado |
|---|---|
| 22/06 | Contratos de rotas, erros, claims e timestamps de migrations acordados; tarefas puxadas. |
| 24/06 | Auth/refresh, clients e middlewares funcionais isoladamente; CI e docs em andamento. |
| 25/06 | Providers/services e OWASP completos; integração end-to-end iniciada. |
| 26/06 | PRs funcionais mesclados. **Feature freeze às 20h.** |
| 27/06 | `INT-01`, testes, CI, README/OpenAPI/Postman e release candidate concluídos. |
| 28/06 | Ensaio, gravação, edição, upload e teste anônimo do vídeo. |
| 29/06 até 18h | Buffer final, tag/release e submissão no SIGAA. |

## Definition of Done

- [ ] Ambiente limpo sobe, aplica migrations e recebe seed por instrução reproduzível.
- [ ] Client e provider cadastram e autenticam; access e refresh são emitidos.
- [ ] Refresh rotaciona, token anterior falha, replay revoga família e logout funciona.
- [ ] Rotas protegidas retornam 401 sem token e roles/ownership retornam 403 quando necessário.
- [ ] CRUD persiste no PostgreSQL via sqlc e o endpoint 1:N retorna dados aninhados.
- [ ] Rate limiting e controle de acesso por objeto são demonstrados e testados como correções OWASP.
- [ ] Pipeline remoto está verde com build, vet, migrations e testes PostgreSQL.
- [ ] README, OpenAPI e Postman refletem a API final.
- [ ] Vídeo cobre a rubrica, dura aproximadamente 12 minutos e o link foi testado anonimamente.
- [ ] Links do repositório e vídeo foram revisados e submetidos no SIGAA.

## Bônus

Somente se o Definition of Done estiver 100% concluído até 27/06: adicionar uma query GraphQL somente de leitura para provider com services, reutilizando repositories e autorização existentes.
