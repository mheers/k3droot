all: build

build:
	docker build -t mheers/k3droot:latest .

push:
	docker push mheers/k3droot:latest
