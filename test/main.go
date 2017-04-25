package main

import (
	"os"

	"github.com/timberio/agent/test"
)

func main() {
	server.AcceptLogs(os.Stdout)
}
