# Data Source: kubectl_server_version

This provider provides a `data` resource `kubectl_server_version` to enable looking up of a kubernetes server version information.
This is particularily helpful if you need to match specific components with the kubernetes server version, e.g. `kube-proxy`.

## Example Usage

```hcl
data "kubectl_server_version" "current" { }
```

## Attribute Reference

* `version` - Version of the server, e.g. `v1.12.10`.
* `major` - Major version, semver if available, e.g. `1`.
* `minor` - Minor version, semver if available, e.g. `12`.
* `patch` - Patch version, semver if available, e.g. `10`.
* `git_version` - Version of the server, e.g. `v1.12.10-eks-aae39f`.
* `git_commit` - Git sha commit, e.g. `aae39f4697508697bf16c0de4a5687d464f4da81`.
* `build_date` - Date server binaries were build, e.g. `2019-12-23T08:19:12Z`.
* `platform` - Server platform name, e.g. `linux/amd64`.
