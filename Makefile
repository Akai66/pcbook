gen:
	protoc --proto_path=proto proto/*.proto --go_out=plugins=grpc:./

clean:
	rm pb/*.go

server1:
	go run cmd/server/main.go -port 50051
server2:
	go run cmd/server/main.go -port 50052

server1-tls:
	go run cmd/server/main.go -port 50051 -tls
server2-tls:
	go run cmd/server/main.go -port 50052 -tls

server:
	go run cmd/server/main.go -port 8080

client:
	go run cmd/client/main.go -address 0.0.0.0:8080

client-tls:
	go run cmd/client/main.go -address 0.0.0.0:8080 -tls


test:
	go test -cover -race ./...

.PHONY: gen clean server client test   #防止命令和项目中文件夹重名,无法执行