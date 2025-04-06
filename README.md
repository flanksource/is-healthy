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


## Attribution

This project builds upon the health check implementations from:
* [Argo CD](https://github.com/argoproj/argo-cd)
* [gitops-engine](https://github.com/argoproj/gitops-engine)
