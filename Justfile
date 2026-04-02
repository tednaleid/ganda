# default recipe shows available commands
default:
    @just --list

# run all checks (CI entry point)
check: test lint fmt-check

# run tests, pass extra args with: just test -v -run TestFoo
test *ARGS:
    go test ./... {{ARGS}}

# run golangci-lint
lint:
    golangci-lint run

# format all go files
fmt:
    gofmt -s -w .

# check formatting (fails if gofmt would change anything)
fmt-check:
    #!/usr/bin/env bash
    set -euo pipefail
    bad=$(gofmt -s -l .)
    if [ -n "$bad" ]; then
        echo "gofmt would change these files:"
        echo "$bad"
        exit 1
    fi

# build the binary
build:
    go build -o ganda

# install to GOPATH/bin
install:
    go install

# clean build artifacts
clean:
    go clean
    rm -f ganda ganda-amd64

# bump version, create annotated tag with release notes, and push to trigger release.
# Usage: just bump 1.0.4
bump version:
    #!/usr/bin/env bash
    set -euo pipefail
    test -n "{{version}}" || { echo "Usage: just bump 1.0.4"; exit 1; }
    tag="v{{version}}"
    # Generate release notes from commits since last tag
    prev_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    if [ -n "$prev_tag" ]; then
        log=$(git log "$prev_tag"..HEAD --oneline --no-merges)
    else
        log=$(git log --oneline --no-merges)
    fi
    notes_file=$(mktemp)
    trap 'rm -f "$notes_file"' EXIT
    if command -v claude >/dev/null 2>&1; then
        claude -p "Generate concise release notes for version {{version}}. Commits:\n$log\n\nGuidelines: group related commits, focus on user-facing changes, skip version bumps and CI changes, one line per bullet, past tense, output only a bullet list." > "$notes_file" 2>/dev/null || echo "$log" | sed 's/^[0-9a-f]* /- /' > "$notes_file"
    else
        echo "$log" | sed 's/^[0-9a-f]* /- /' > "$notes_file"
    fi
    echo "Release notes:"
    cat "$notes_file"
    git tag -a "$tag" -F "$notes_file"
    git push && git push --tags
    echo "$tag released!"

# delete a GitHub release and re-tag to re-trigger release workflow.
# Preserves the annotated tag message (release notes).
# Usage: just retag 1.0.4
retag version:
    #!/usr/bin/env bash
    set -euo pipefail
    tag="v{{version}}"
    # Save existing tag annotation before deleting
    notes=$(git tag -l --format='%(contents)' "$tag" 2>/dev/null || echo "$tag")
    notes_file=$(mktemp)
    trap 'rm -f "$notes_file"' EXIT
    echo "$notes" > "$notes_file"
    gh release delete "$tag" --yes || true
    git push origin ":refs/tags/$tag" || true
    git tag -d "$tag" || true
    git tag -a "$tag" -F "$notes_file"
    git push && git push --tags

# install git pre-commit hook that runs all checks before each commit
install-hooks:
    #!/usr/bin/env bash
    set -euo pipefail
    hook=".git/hooks/pre-commit"
    cat > "$hook" << 'HOOK'
    #!/bin/sh
    just check
    HOOK
    chmod +x "$hook"
    echo "Installed pre-commit hook: $hook"
