# k3droot

> A simple tool to exec into pods in a k3d cluster as root

Based on this [gist](https://gist.github.com/mamiu/4944e10305bc1c3af84946b33237b0e9).

## Dependencies
- docker
- k3d

## Installation
### Binary
```bash
go install github.com/mheers/k3droot@latest
```
### Docker
```bash
docker run --rm -it -v /var/run/docker.sock:/var/run/docker.sock -v $HOME/.kube:/root/.kube/:ro --network host mheers/k3droot:latest
```

## TODO
- [ ] remove dependency of `docker exec`
- [ ] add support for containers that run a from scratch image or do not have a shell (`sh`)
