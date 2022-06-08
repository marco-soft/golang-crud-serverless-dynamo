go_apps = bin/read bin/create bin/update bin/delete bin/read_structure

bin/% : form/*/%.go
		env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o $@ $<

bin/% : structure/*/%.go
		env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o $@ $<

build: $(go_apps)