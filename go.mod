module syncing

go 1.12

require (
	github.com/golang/protobuf v1.3.2
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4 // indirect
)

replace (
	github.com/golang/protobuf v1.3.2 => ./pkg/github.com/golang/protobuf@v1.3.2
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2 => ./pkg/golang.org/x/crypto@v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4 => ./pkg/golang.org/x/crypto@v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/net v0.0.0-20190404232315-eb5bcb51f2a3 => ./pkg/golang.org/x/net@v0.0.0-20190404232315-eb5bcb51f2a3
	golang.org/x/sys v0.0.0-20190412213103-97732733099d => ./pkg/golang.org/x/sys@v0.0.0-20190412213103-97732733099d
	golang.org/x/text v0.3.0 => ./pkg/golang.org/x/text@v0.3.0
)
