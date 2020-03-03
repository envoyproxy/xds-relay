# Go coding style

* The xDS Relay source code is formatted using
  [golangci-lint](https://github.com/golangci/golangci-lint). Thus all white spaces, etc.
  issues are taken care of automatically. Azure pipeline tests will automatically check the code
  format and fail. The specific linters used can be found [here](.golangci.yml).
* More detailed descriptions of each linter can be found
  [here](https://github.com/golangci/golangci-lint/tree/v1.23.6#enabled-by-default-linters).

# Recommended Go style guides

* [Effective Go](https://golang.org/doc/effective_go.html)
* [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
