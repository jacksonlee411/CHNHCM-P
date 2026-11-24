#!/usr/bin/env bash
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
fixture_dir=$(mktemp -d)
trap 'rm -rf -- "$fixture_dir"' EXIT
for version in 26 27; do
  case "$version" in
    26) release=9c490a97; quota=0 ;;
    27) release=b647031b; quota=1 ;;
  esac
  src="$fixture_dir/source$version"
  backup="$fixture_dir/backup$version"
  mkdir "$src" "$backup"
  git archive "$release" go.mod go.sum cmd internal db deploy/local/postgres-image.txt | tar -x -C "$src"
  cp tests/integration/testdata/personnel/legacy_backup.go.in "$src/internal/bootstrap/position_backup_integration_test.go"
  (
    cd "$src"
    CHNHCM_BACKUP_POSITION_QUOTA="$quota" PERSONNEL_BACKUP_DIR="$backup" \
      go test -tags=integration ./internal/bootstrap -run '^TestBackupBeforePersonnel$' -count=1
  )
done
CHNHCM_STAFFING_BACKUP_DIR="$fixture_dir" go test -tags=integration ./internal/bootstrap -run '^TestHire400PositionBackups$' -count=1 -v
