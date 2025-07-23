simple:
	cd delve_wrapper && go build -o ../tdlv.build && cd ../example/simple-workflow && ../../tdlv.build

structured:
	cd delve_wrapper && go build -o ../tdlv.build && cd ../example/structured-workflow/replay-debug && ../../../tdlv.build

build:
	cd delve_wrapper && go build -o ../tdlv.build