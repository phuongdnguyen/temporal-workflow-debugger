simple:
	cd custom-debugger && go clean && go build -o ../tdlv.build && cd ../example/simple-workflow && ../../tdlv.build

structured-ide-integrated:
	cd custom-debugger && go clean && go build -o ../tdlv.build && cd ../example/structured-workflow/replay-debug-ide-integrated && ../../../tdlv.build

structured-standalone:
	cd custom-debugger && go clean && go build -o ../tdlv.build && cd ../example/structured-workflow/replay-debug-standalone && ../../../tdlv.build

build:
	cd custom-debugger && go clean && go build -o ../tdlv.build

run-ide:
	cd jetbrains-plugin &&  ./gradlew runIde