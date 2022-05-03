.PHONY: ui server pre-build-api all run

all: server

ui:
	cd ui && yarn && yarn build

pre-build-api: ui
	cp -r ui/build server/api/

server: pre-build-api
	cd server && go build

run:
	cd server && ./server

edit-config:
	vim server/.config.yaml
