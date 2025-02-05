test:
	go run . testdata/sample.go testdata/sample_gopack.go
	go test . -v -count=1
