package netway

import (
	"testing"
	"zlib"
)

func TestNewPlayerManager(t *testing.T) {
	PlayerManager := NewPlayerManager(0,0,0)
	zlib.MyPrint(PlayerManager)
	t.Log("okkkkkk")
}