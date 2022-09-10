
test-build:
	@for os in linux darwin windows; do \
		GOOS=$$os go build .;\
	done
