all: docker binary

docker:
	docker build -t dyshard:latest .

binary:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dyshard .