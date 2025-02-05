test:
	go run . testdata/sample.go
	go test . -v -count=1
