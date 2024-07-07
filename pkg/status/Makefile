tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

generate:
	protoc --proto_path=. --go_out=. cluster_status.proto 