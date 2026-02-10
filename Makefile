default: build

update-gfwlist:
	go run ./cmd/gfwlist2pac -out gfwlist.pac

build:
	@CGO_ENABLED=0 go build -o pac-server .
