module syncing

go 1.12

require (
	github.com/golang/protobuf v1.3.2
	github.com/pkg/profile v1.3.0
)

replace (
	github.com/golang/protobuf v1.3.2 => ./pkg/github.com/golang/protobuf@v1.3.2
	github.com/pkg/profile v1.3.0 => ./pkg/github.com/pkg/profile@v1.3.0
)
