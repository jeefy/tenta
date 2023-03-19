build:
	go build -o bin/tenta *.go

run:
	./bin/tenta

image:
	docker build -t jeefy/tenta .

image-push:
	docker push jeefy/tenta