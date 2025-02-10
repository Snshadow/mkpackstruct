test:
	go run . testdata/sample.go testdata/sample_packstruct.go
	go test . -v -count=1
