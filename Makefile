BINARY_PRE=*.go server/*.go css/main.css server/bindata.go
BINARY_SOURCE=*.go

BINDATA_DATA=css/main.css static/* templates/* img/*
BINDATA_FLAGS=-pkg server
BINDATA_FLAGS_DEBUG=-pkg server -debug

SASS_PRE=sass/*.scss
SASS_SOURCE=sass/main.scss
SASS_FLAGS=-t compressed
SASS_FLAGS_DEBUG=-t nested -l

build: bin/hashworksNET

generate: $(BINARY_PRE)

run: build
	bin/hashworksNET

distribute: build
	./distribute.sh

debug: SASS_FLAGS=$(SASS_FLAGS_DEBUG)
debug: BINDATA_FLAGS=$(BINDATA_FLAGS_DEBUG)
debug: bin/hashworksNET
	bin/hashworksNET --debug

dependencies:
	go get -u github.com/gin-gonic/gin
	go get -u github.com/ekyoung/gin-nice-recovery
	go get -u github.com/gin-contrib/multitemplate
	go get -u github.com/gin-contrib/cache
	gp get -u github.com/gin-gonic/autotls
	go get -u github.com/unrolled/secure
	go get -u github.com/jteeuwen/go-bindata/...
	go get -u github.com/elazarl/go-bindata-assetfs/...
	go get -u github.com/mattn/go-isatty
	go get -u github.com/wcharczuk/go-chart

clean:
	rm -Rf ./bin ./css ./server/bindata.go


debug-css: SASS_FLAGS=$(SASS_FLAGS_DEBUG)
debug-css: css

css/main.css: $(SASS_PRE)
	mkdir -p css
	sassc $(SASS_FLAGS) $(SASS_SOURCE) $@


debug-bindata: BINDATA_FLAGS=$(BINDATA_FLAGS_DEBUG)
debug-bindata: server/bindata.go

server/bindata.go: $(BINDATA_DATA)
	go-bindata $(BINDATA_FLAGS) -o $@ $(BINDATA_DATA)


bin/hashworksNET: $(BINARY_PRE)
	mkdir -p bin
	go build -o bin/hashworksNET $(BINARY_SOURCE)
