simple:
	cd custom-debugger && go clean && go build -o ../tdlv.build && cd ../example/simple-workflow && ../../tdlv.build

structured:
	cd custom-debugger && go clean && go build -o ../tdlv.build && cd ../example/structured-workflow/replay-debug-ide-integrated && ../../../tdlv.build

build:
	cd custom-debugger && go clean && go build -o ../tdlv.build

run-ide:
	cd jetbrains-plugin &&  ./gradlew runIde