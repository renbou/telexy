.PHONY:	test
test:
	go test -race -bench=. -benchmem -count=1 ./...

.PHONY:	race-test
race-test:
	go test -race -cpu 1,8,100 -count=1 -run='^TestMux$$' ./...

.PHONY: bench
bench:
	go test -run='^$$' -bench=. -benchmem ./...

.PHONY: cover
cover:
	go test -race -count=1 \
		-bench=. -benchmem \
		-covermode=atomic -coverprofile cover.tmp.out -coverpkg=./... \
		./... && \
	grep -v 'genproto\|mocks' cover.tmp.out > cover.out
	go tool cover -func cover.out && \
	go tool cover -html cover.out -o cover.html && \
	open cover.html && sleep 1 && \
	rm -f cover.tmp.out cover.out cover.html

.PHONY: lint
lint:
	golangci-lint run -v