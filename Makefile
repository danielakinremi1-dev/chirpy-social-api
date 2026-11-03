serve:
	go build -o out ./cmd/chirpy && ./out

run:
	go run ./cmd/chirpy

up:
	goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/chirpy" up

down:
	goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/chirpy" down

chirp:
	sudo -u postgres psql chirpy

gen:
	sqlc generate