.PHONY: up down build prod delete alldelete front back db front_build

include .env

FRONT_CONTAINER_ID=$(shell docker compose ps -q frontend)
BACK_CONTAINER_ID=$(shell docker compose ps -q backend)
DB_CONTAINER_ID=$(shell docker compose ps -q db)

up:
	docker compose up
down:
	docker compose down
build:
	docker compose build
prod:
	docker compose -f docker-compose.prod.yaml up
delete:
	docker compose down --rmi all --remove-orphans
alldelete:
	docker compose down --rmi all --volumes --remove-orphans
front:
	docker exec -it $(FRONT_CONTAINER_ID) bash
back:
	docker exec -it $(BACK_CONTAINER_ID) sh
db:
	docker exec -it $(DB_CONTAINER_ID) bash -c "mysql -u $(MYSQL_USER) -p"
front_build:
	docker compose run --rm frontend yarn vite build