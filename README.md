# vcluster-vnode-plugin

This plugin is intended to make vCluster work correctly with [`github.com/kroderdev/vnode`](https://github.com/kroderdev/vnode).

## Why it exists

When a pod inside vCluster is assigned to a vnode, its `spec.nodeName` points to a virtual node name like `vnode-*`.

That node name does not exist as a real schedulable host node, so sending it unchanged to the host cluster causes problems.

This plugin fixes that by:

- clearing vnode-based `spec.nodeName` before the host pod is created
- storing the original vnode name in `vnode.kroderdev.io/node-name`
- restoring that vnode name when the host pod is read back into vCluster

This allows the host cluster to schedule onto a real node while vCluster still sees the pod as bound to the vnode.

## Behavior

For pods with `spec.nodeName` starting with `vnode-`:

- on create: move the node name into a label and clear `spec.nodeName`
- on get: restore `spec.nodeName` from the label

Pods without a vnode-style node name are left unchanged.

## Development

Run tests:

```bash
go test ./... -v -coverprofile=coverage.out -covermode=atomic
```

Build:

```bash
go build -o bin/
```

Build image:

```bash
docker build -t ghcr.io/kroderdev/vcluster-vnode-plugin:dev -f Dockerfile .
```
