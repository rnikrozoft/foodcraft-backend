.PHONY: build run stop logs rebuild

build:
	docker build -t foodcraft-backend-plugin .
	docker create --name foodcraft-backend-temp foodcraft-backend-plugin /bin/true
	docker cp foodcraft-backend-temp:/backend.so ./modules/backend.so
	docker rm foodcraft-backend-temp

run:
	docker compose up -d

stop:
	docker compose down

logs:
	docker compose logs -f nakama

rebuild: build run
