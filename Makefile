postgres:
	docker run --name postgres15 -p 5430:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=banach -d postgres:latest

createdb:
	docker exec -it postgres15 createdb --username=root --owner=root fcn

dropdb:
	docker exec -it postgres15 dropdb fcn

migrateup:
	migrate -path db/migration -database "postgresql://root:banach@localhost:5430/fcn?sslmode=disable" -verbose up

migratedown:
	migrate -path db/migration -database "postgresql://root:banach@localhost:5430/fcn?sslmode=disable" -verbose down

sqlc:
	sqlc generate

server:
	go run main.go

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/banachtech/spotted-zebra/db/sqlc Store

.PHONY: postgres createdb dropdb migrateup migratedown sqlc server mock