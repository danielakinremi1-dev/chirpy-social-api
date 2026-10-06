serve:
	go build -o out && ./out

run:
	go run .

up:
	goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/chirpy" up

down:
	goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/chirpy" down

chirp:
	sudo -u postgres psql chirpy

gen:
	sqlc generate