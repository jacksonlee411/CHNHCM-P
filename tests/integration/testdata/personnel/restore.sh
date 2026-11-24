#!/usr/bin/env bash
set -euo pipefail
# Build the previous product in isolation and keep the synthetic credentials outside the worktree.
cd "$(git rev-parse --show-toplevel)"
previous=f7d84f192ff518555a60dfe939a78455d8a48597
temp_dir=$(mktemp -d)
trap 'rm -rf -- "$temp_dir"' EXIT
mkdir "$temp_dir/source" "$temp_dir/backup"
git archive "$previous" go.mod go.sum cmd internal db deploy/local/postgres-image.txt | tar -x -C "$temp_dir/source"
cp tests/integration/testdata/personnel/legacy_backup.go.in "$temp_dir/source/internal/bootstrap/personnel_backup_integration_test.go"
(
  cd "$temp_dir/source"
  PERSONNEL_BACKUP_DIR="$temp_dir/backup" go test -tags=integration ./internal/bootstrap -run '^TestBackupBeforePersonnel$' -count=1
)
CHNHCM_PERSONNEL_BACKUP_DIR="$temp_dir/backup" go test -tags=integration ./internal/bootstrap -run '^TestPersonnelUpgradeFromBackup$' -count=1
