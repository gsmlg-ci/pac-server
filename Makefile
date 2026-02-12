default: build

download:
	curl -fsSL https://raw.githubusercontent.com/gfwlist/gfwlist/refs/heads/master/gfwlist.txt -o gfwlist.txt

update-gfwlist: download

build:
	@CGO_ENABLED=0 go build -o pac-server .
