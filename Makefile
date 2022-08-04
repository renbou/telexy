.PHONY: test
test:
	go test -v -race -bench=. -benchmem -count=1 ./...

.PHONY: bench
bench:
	go test -v -run='^$$' -bench=. -benchmem ./...

.PHONY: cover
cover:
	go test -race -bench=. -benchmem -count=1 -covermode=atomic -coverprofile cover.tmp.out -coverpkg=./... -v ./... && \
	grep -v 'genproto\|mocks' cover.tmp.out > cover.out
	go tool cover -func cover.out && \
	go tool cover -html cover.out -o cover.html && \
	open cover.html && sleep 1 && \
	rm -f cover.tmp.out cover.out cover.html

.PHONY: lint
lint:
	golangci-lint run -v