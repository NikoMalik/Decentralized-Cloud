build:
	@go build -o  bin/cloud 


run: build
	@./bin/cloud


clean:
	@rm -rf bin/cloud


tidy:
	@go mod tidy -v

buildWindows:
	@go build -o bin/cloud.exe


runWindows: buildWindows
	@./bin/cloud.exe


buildMacos:
	@go build -o bin/cloud_osx


runMacos: buildMacos
	@./bin/cloud_osx


test:
	@go test -timeout 30s  -v ./testing

bench:
	@go test -benchmem   -timeout 10s -bench . -benchmem github.com/NikoMalik/Decentralized-Cloud/benchmarks


.PHONY: build run test bench


buildAll: build buildWindows buildMacos

cleanAll:
	@rm -rf bin/*