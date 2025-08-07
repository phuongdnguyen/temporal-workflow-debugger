go-simple:
	cd tdlv && go clean && go build -o ../tdlv.build && cd ../example/go/simple-workflow && ../../../tdlv.build --lang=go

go-structured-ide-integrated:
	cd tdlv && go clean && go build -o ../tdlv.build && cd ../example/go/structured-workflow/replay-debug-ide-integrated && ../../../../tdlv.build --lang=go

go-structured-standalone:
	cd tdlv && go clean && go build -o ../tdlv.build && cd ../example/go/structured-workflow/replay-debug-standalone && ../../../../tdlv.build --lang=go

python:
	cd tdlv && go clean && go build -o ../tdlv.build && cd .. && ./tdlv.build --lang=python

# Node.js/TypeScript replayer
js:
	cd tdlv && go clean && go build -o ../tdlv.build && cd ../example/js && ../../tdlv.build --lang=js

build:
	cd tdlv && go clean && go build -o ../tdlv.build

run-ide:
	cd jetbrains-plugin &&  ./gradlew runIde