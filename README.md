<div align="center">
  <p>
    <a href="https://github.com/flanksource/is-healthy/actions"><img src="https://github.com/flanksource/is-healthy/workflows/Test/badge.svg"></a>
    <a href="https://goreportcard.com/report/github.com/flanksource/is-healthy"><img src="https://goreportcard.com/badge/github.com/flanksource/is-healthy"></a>
    <img src="https://img.shields.io/github/license/flanksource/is-healthy.svg?style=flat-square"/>
  </p>
</div>
`is-healthy` is a heuristic library for checking the health of Kubernetes and Cloud resources, it supports basic, enhanced and heuristic mode.


* **Basic Mode**: Wraps Argo health check functionality. Does not distinguish between `health` and `status`, `lastUpdated` and `ready` are not supported
  * **health**: One of `healthy`, `unhealthy`, `warning`, `unknown`
  * **message**: A reason for the status e.g. `Back-off pulling image "nginx:invalid"`
* **Enhanced Mode**: returns the following fields:
  
  * ***ready***: Whether the resource is currently being reconciled / provisioned.
  > `ready`  is independent of `health`, e.g. A pod in a failure state can be ready if its state is terminal and will not change.
  * **health**: One of `healthy`, `unhealthy`, `warning`, `unknown`,
  > `health` can transition based on the age or last event time of a resource.
  * **status**: A text description of the state of of resource e.g. `Running` or `ImagePullBackoff`
  * **message**: A reason for the status e.g. `Back-off pulling image "nginx:invalid"`
  * **lastUpdated**: The last time the resource was updated
* **Heuristic Mode**: Attempts to determine health using and fields named `state`, `status` etc..

|Object Type|Support Mode||
|---|---|---|
|Core Kubernetes Resources|Enhanced||
|Pod|Enhanced|Ignores pod restarts for the first `15m` <br />`warning` if restarted in in last 24h<br />`unhealthy` if restarted in last `1h`|
|Certificate|Enhanced|`unhealthy` if not issued with `1h`<br /> `warning` if not issued with `15m` <br/>`warning` if certificate expiry  `< 2d`|
|CronJob|Enhanced||
|Flux CRD's|Enhanced||
|Argo CRD's|Enhanced||
|Cert-Manager CRD's|Enhanced|Marks|
|Kubernetes Resources using [Conditions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties)|Heuristic||




## Example

```shell
kubectl po  -o json <pod-name> | is-healthy -j
```

Output:

```json
{
  "ready": false,
  "health": "unhealthy",
  "status": "ImagePullBackOff",
  "message": "Back-off pulling image \"nginx:invalid\"",
  "lastUpdated": "2025-03-26T10:17:18Z"
}
```


## Optional property updates

Health checks use their built-in defaults unless an application opts into property updates. Applications using `github.com/flanksource/commons/properties` can enable the listener during startup, before loading or updating properties:

```go
import (
    "github.com/flanksource/commons/properties"
    "github.com/flanksource/is-healthy/pkg/health"
)

func configureHealth() {
    health.RegisterPropertyListener(properties.RegisterListener)
}
```

This replaces the previously automatic registration on importing `health`. Registration installs the callback; settings are applied when properties change. An individual properties instance's `RegisterListener` method also works. Applications enable the listener before concurrent health checks start.

| Property | Default | Purpose |
|----------|---------|---------|
| `health.cert-manager.expiryGracePeriod` | `48h` | Warn when a certificate expires within this period. |
| `health.cert-manager.renewalGracePeriod` | `30m` | Control overdue renewal and issuance thresholds. |
| `health.azure.clientSecretExpiryGracePeriod` | `720h` | Warn when an Azure credential expires within this period. |
| `health.maxMessageLength` | `512` | Limit messages where health checks truncate individual errors. |

Removing a property restores its default on the next update. A zero value retains the current setting, preserving the previous listener behavior. The core library does not depend on commons; the caller owns that optional dependency.

## Maintaining copied dependencies

The small source subsets in `internal/certmanager`, `internal/lo`, and `internal/duration` are maintained manually. Their headers identify the upstream files, pinned source versions, and adaptations; license files sit alongside them. When refreshing a copy, review the attributed source and retain only the health-related declarations. The `lo.Capitalize` copy retains `golang.org/x/text` to preserve Unicode title casing.

Install [Task](https://taskfile.dev/) and the [flanksource/gopatch fork](https://github.com/flanksource/gopatch) with its `patch` command, then use:

```shell
task deps:preview
task deps:rewrite
go mod tidy
task deps:check
task deps:report
```

The rewrite task only migrates imports; it does not fetch or regenerate copied source. The report counts the module graph and the modules compiled by `go test ./...` separately, including is-healthy itself. Equivalent Make targets are `deps-preview`, `deps-rewrite`, `deps-check`, and `deps-report`.

## Attribution

This project builds upon the health check implementations from:
* [Argo CD](https://github.com/argoproj/argo-cd)
* [gitops-engine](https://github.com/argoproj/gitops-engine)
