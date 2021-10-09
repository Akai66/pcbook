gen:
	protoc --proto_path=proto proto/*.proto --go_out=plugins=grpc:./

clean:
	rm pb/*.go

server:
	go run cmd/server/main.go -port 8080

client:
	go run cmd/client/main.go -address 0.0.0.0:8080

test:
	go test -cover -race ./...

.PHONY: gen clean server client test   #防止命令和项目中文件夹重名,无法执行