PROTO_DIR = proto
OUT_DIR = internal/inbound/server/pb

.PHONY: proto clean

proto: clean
	echo "Generating Go code from proto files..."
	protoc --proto_path=$(PROTO_DIR) \
	--go_out=$(OUT_DIR) \
	--go-grpc_out=$(OUT_DIR) \
	--go_opt=paths=source_relative \
	--go-grpc_opt=paths=source_relative \
	$(PROTO_DIR)/*.proto

clean:
	echo "Cleaning generated files..."
	rm -rf $(OUT_DIR)/*.pb.go