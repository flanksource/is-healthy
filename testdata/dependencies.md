---
cwd: ..
timeout: 600s
build: make build
---

# Dependency reduction

| Name | Command | Exit Code | CEL Validation |
|------|---------|-----------|----------------|
| Go tests | go test ./... | 0 | stdout.contains("pkg/health") && stdout.contains("pkg/lua") |
| Lint | make lint | 0 | exitCode == 0 |
| Removed modules | go list -m all | 0 | !stdout.contains("github.com/cert-manager/cert-manager ") && !stdout.contains("github.com/flanksource/commons ") && !stdout.contains("github.com/samber/lo ") && !stdout.contains("golang.org/x/exp ") |
| Pod smoke | ./.bin/is-healthy -j -f pkg/health/testdata/Kubernetes/Pod/healthy.yaml | 0 | json.health == "healthy" |
| Expired certificate smoke | ./.bin/is-healthy -j -f pkg/health/testdata/certificate-expired.yaml | 1 | json.health == "unhealthy" && json.status == "Expired" |
| Repeatable import migration | task deps:check | 0 | exitCode == 0 |
