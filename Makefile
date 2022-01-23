WEBAPP=ishocon@54.178.150.68
BENCH=ishocon@52.197.190.213

build: ishocon1-go

deploy: ishocon1-go
	scp ishocon1-go $(WEBAPP):ISHOCON1/webapp/go/
	rsync -av ./templates/ $(WEBAPP):ISHOCON1/webapp/go/templates/

ishocon1-go: main.go
	go build -o ishocon1-go
