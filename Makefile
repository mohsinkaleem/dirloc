VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY     := dirloc
LDFLAGS    := -ldflags "-X github.com/dirloc/dirloc/cmd.Version=$(VERSION)"
PROFILE_PATH ?= .

.PHONY: build install test test-short bench stress lint clean \
        profile-cpu profile-mem profile-all pprof-cpu pprof-mem

build:
	go build $(LDFLAGS) -o $(BINARY) .

install:
	go install $(LDFLAGS) .

test:
	go test ./... -v

test-short:
	go test ./... -short

# Run all benchmarks and print allocations
bench:
	go test ./... -bench=. -benchmem -run=^$$ -benchtime=3s

# Run only the stress tests (skipped by default in `make test`)
stress:
	go test ./... -v -run=Stress -timeout=120s

lint:
	golangci-lint run ./...

# --- profiling targets -------------------------------------------------------

# Capture a CPU profile while scanning PROFILE_PATH.
# Usage: make profile-cpu PROFILE_PATH=~/myrepo
profile-cpu: build
	./$(BINARY) $(PROFILE_PATH) --lang --cpuprofile cpu.prof
	@echo "CPU profile written to cpu.prof — run: make pprof-cpu"

# Capture a heap/memory profile while scanning PROFILE_PATH.
profile-mem: build
	./$(BINARY) $(PROFILE_PATH) --lang --memprofile mem.prof
	@echo "Memory profile written to mem.prof — run: make pprof-mem"

# Capture both profiles in one pass.
profile-all: build
	./$(BINARY) $(PROFILE_PATH) --lang --cpuprofile cpu.prof --memprofile mem.prof
	@echo "Profiles written to cpu.prof and mem.prof"

# Open the CPU profile in an interactive pprof shell.
pprof-cpu:
	go tool pprof -http=:6060 cpu.prof

# Open the memory profile in an interactive pprof shell.
pprof-mem:
	go tool pprof -http=:6060 mem.prof

clean:
	rm -f $(BINARY) cpu.prof mem.prof
