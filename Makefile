PROJECT_NAME=gotil

dev: dev-crud-server

dev-crud-server:
	cd apps/crud-server && \
	make dev

dev-goup:
	cd apps/goup && \
	make dev