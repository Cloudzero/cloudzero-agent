package build

import (
	"github.com/go-obvious/server"
)

func Version() *server.ServerVersion {
	return &server.ServerVersion{Revision: Rev, Tag: Tag, Time: Time}
}
