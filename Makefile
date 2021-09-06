build:
	go build -o bin/tenta *.go

run:
	go run *.go

image:
	docker build -t jeefy/tenta .

image-push:
	docker push jeefy/tenta