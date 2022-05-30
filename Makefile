go_apps = bin/read bin/create bin/update bin/delete

bin/% : form/*/%.go
		env GOOS=linux go build -ldflags="-s -w" -o $@ $<

build: $(go_apps)