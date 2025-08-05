go-simple:
	cd custom-debugger && go clean && go build -o ../tdlv.build && cd ../example/go/simple-workflow && ../../../tdlv.build --lang=go

go-structured-ide-integrated:
	cd custom-debugger && go clean && go build -o ../tdlv.build && cd ../example/go/structured-workflow/replay-debug-ide-integrated && ../../../../tdlv.build --lang=go

go-structured-standalone:
	cd custom-debugger && go clean && go build -o ../tdlv.build && cd ../example/go/structured-workflow/replay-debug-standalone && ../../../../tdlv.build --lang=go

python:
	cd custom-debugger && go clean && go build -o ../tdlv.build && cd .. && ./tdlv.build --lang=python

# Node.js/TypeScript replayer
js:
	cd custom-debugger && go clean && go build -o ../tdlv.build && cd ../example/js && ../../tdlv.build --lang=js

build:
	cd custom-debugger && go clean && go build -o ../tdlv.build

run-ide:
	cd jetbrains-plugin &&  ./gradlew runIde