module frame_sync

go 1.14

require (
	zlib v0.0.0
	github.com/gorilla/websocket v1.4.2
)

replace (
	zlib v0.0.0 => ../zlib
)
