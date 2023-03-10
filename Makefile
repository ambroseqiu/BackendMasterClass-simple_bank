DB_URL = postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable
postgres:
	docker run --name postgres12 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:12-alpine

createdb:
	docker exec -it postgres12 createdb --username=root --owner=root simple_bank

dropdb:
	docker exec -it postgres12 dropdb simple_bank

migrateup:
	migrate -path db/migration -database "${DB_URL}" -verbose up

migratedown:
	migrate -path db/migration -database "${DB_URL}" -verbose down

migrateup1:
	migrate -path db/migration -database "${DB_URL}" -verbose up 1

migratedown1:
	migrate -path db/migration -database "${DB_URL}" -verbose down 1

server:
	go run main.go
sqlc:
	sqlc generate
test:
	go test -v -cover ./...
mock:
	mockgen -package mockdb -destination=./db/mock/store.go github.com/backendmaster/simple_bank/db/sqlc Store
proto:
	rm -f pb/*.go
	rm -f docs/swagger/*.swagger.json
	protoc --proto_path=proto --go_out=pb --go_opt=paths=source_relative \
    --go-grpc_out=pb --go-grpc_opt=paths=source_relative \
	--grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative \
	--openapiv2_out=docs/swagger --openapiv2_opt=allow_merge=true,merge_file_name=simple_bank \
    proto/*.proto
evans:
	evans --host localhost --port 9090 -r repl

.PHONY: postgres createdb dropdb migratieup migratiedown migratieup1 migratiedown1 sqlc test server mock proto evans