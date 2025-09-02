example: example.go tinybox/tb.go
	go build -o example example.go

clean:
	rm -f example

install: example
	cp example /usr/local/bin/

.PHONY: clean install
