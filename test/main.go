package main

import (
	"os"

	"github.com/timberio/agent/test/server"
)

func main() {
	server.AcceptLogs(os.Stdout)
}
