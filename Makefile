simple:
	export LANGUAGE=go && cd custom-debugger && go clean && go build -o ../tdlv.build && cd ../example/simple-workflow && ../../tdlv.build

structured-ide-integrated:
	cd custom-debugger && go clean && go build -o ../tdlv.build && cd ../example/structured-workflow/replay-debug-ide-integrated && ../../../tdlv.build

structured-standalone:
	cd custom-debugger && go clean && go build -o ../tdlv.build && cd ../example/structured-workflow/replay-debug-standalone && ../../../tdlv.build

debugpy:
	cd example/python && python -m debugpy --listen 2345 --wait-for-client standalone_replay.py

python:
	export LANGUAGE=python && cd custom-debugger && go clean && go build -o ../tdlv.build && cd .. && ./tdlv.build

build:
	cd custom-debugger && go clean && go build -o ../tdlv.build

run-ide:
	cd jetbrains-plugin &&  ./gradlew runIde