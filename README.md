# k3droot

> A simple tool to exec into pods in a k3d cluster as root

Based on this [gist](https://gist.github.com/mamiu/4944e10305bc1c3af84946b33237b0e9).

## Dependencies
- docker
- k3d

## Installation
```bash
go install github.com/mheers/k3droot
```

## TODO
- [ ] remove dependency of `docker exec`
- [ ] add support for containers that run a from scratch image or do not have a shell (`sh`)
