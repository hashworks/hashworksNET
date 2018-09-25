BINARY_PRE=*.go server/*.go css/main.css server/bindata/bindata.go
BINARY_SOURCE=*.go

BINDATA_DATA=css/main.css static/pgp_public_key.asc templates/* img/*
BINDATA_FLAGS=-pkg bindata
BINDATA_FLAGS_DEBUG=-pkg bindata -debug

SASS_PRE=sass/*.scss
SASS_SOURCE=sass/main.scss
SASS_FLAGS=-t compressed
SASS_FLAGS_DEBUG=-t nested -l

build: bin/hashworksNET

run: build
	bin/hashworksNET

distribute: build
	./distribute.sh

clean:
	rm -Rf ./bin ./css ./server/bindata.go

debug: SASS_FLAGS=$(SASS_FLAGS_DEBUG)
debug: BINDATA_FLAGS=$(BINDATA_FLAGS_DEBUG)
debug: bin/hashworksNET
	bin/hashworksNET --debug


debug-css: SASS_FLAGS=$(SASS_FLAGS_DEBUG)
debug-css: css

css/main.css: $(SASS_PRE)
	mkdir -p css
	sassc $(SASS_FLAGS) $(SASS_SOURCE) $@


debug-bindata: BINDATA_FLAGS=$(BINDATA_FLAGS_DEBUG)
debug-bindata: server/bindata.go

server/bindata/bindata.go: $(BINDATA_DATA)
	go-bindata $(BINDATA_FLAGS) -o $@ $(BINDATA_DATA)


bin/hashworksNET: $(BINARY_PRE)
	mkdir -p bin
	go build -o bin/hashworksNET $(BINARY_SOURCE)
