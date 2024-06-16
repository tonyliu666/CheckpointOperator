gen:
	cd ./proto && protoc --go_out=../generated --go_opt=paths=source_relative --go-grpc_out=../generated --go-grpc_opt=paths=source_relative *.proto
clean: 
	rm pb/*.go 
server: 
	go run cmd/server/main.go -port 50051
client:
	go run cmd/client/main.go -connect localhost:50051 -archiveDir /home/Tony/MyOperstorProjects/practice/grpc/tar-file/nginx.tar -podName nginx -checkpointName nginx.tar
clang-format: 
	clang-format -i ./proto/*.proto
test-cover: 
	go test -cover -race ./...