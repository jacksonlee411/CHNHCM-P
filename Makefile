PNPM ?= corepack pnpm

.PHONY: install dev server build check docs boundaries runtime-tests smoke integration ci-preflight ci-preview ci-ready load local-up local-down generate
install:
	$(PNPM) install --frozen-lockfile
dev:
	$(PNPM) dev
server:
	go run ./cmd/server
build:
	mkdir -p .local/bin
	go build -o .local/bin/server ./cmd/server
	go build -o .local/bin/upgrader ./cmd/upgrader
	$(PNPM) build
docs:
	python3 -B -m unittest discover -s scripts -p 'test_*.py'
	python3 -B scripts/check-repository.py
boundaries:
	go run ./scripts/check-boundaries
runtime-tests:
	node --test scripts/test_*.mjs
check: docs boundaries runtime-tests
	python3 scripts/generate-name-tables.py --check
	go run ./internal/contexts/organization/infra/history/generate --check
	go test ./...
	go vet ./...
	$(PNPM) typecheck
	git diff --check
smoke:
	python3 scripts/local-bu.py smoke $(if $(SMOKE_PROFILE),--profile $(SMOKE_PROFILE)) -- $(SMOKE_ARGS)
integration:
	go test -tags=integration -timeout=20m ./tests/integration ./internal/bootstrap -count=1
ci-preview:
	python3 -B scripts/ci_preview.py --base "$${CI_BASE:-origin/main}" $${CI_HEAD:+--head "$$CI_HEAD"}
ci-ready:
	python3 -B scripts/ci_preview.py --base "$${CI_BASE:-origin/main}" --ready
ci-preflight:
	python3 -B scripts/ci_preview.py --base HEAD --preflight
load:
	CHNHCM_LOAD=1 go test -tags=integration ./tests/integration -run TestBUSliceLoad -v -count=1
local-up:
	python3 scripts/local-bu.py up
local-down:
	python3 scripts/local-bu.py down
generate:
	python3 scripts/generate-name-tables.py
	go run ./internal/contexts/organization/infra/history/generate
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
	$(PNPM) generate:api
