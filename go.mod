module github.com/Fisch-Labs/FishDB

go 1.25.0

require (
	github.com/Fisch-Labs/Tide v1.0.0
	github.com/Fisch-Labs/Toolkit v1.0.0
	github.com/gorilla/websocket v1.4.1
)

replace (
	github.com/Fisch-Labs/Tide v1.0.0 => ./external_deps/Tide
	github.com/Fisch-Labs/Toolkit v1.0.0 => ./external_deps/Toolkit
)
