package middlewares

import (
	"bytes"
	"io"
	"log"
	"os"
)

// authzFilterWriter drops noisy authz debug logs emitted by the auth-manager dependency
// while preserving all other standard log output.
type authzFilterWriter struct {
	base io.Writer
}

func (w authzFilterWriter) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte("[AUTHZ")) {
		// Pretend we wrote the bytes so upstream logging doesn't error
		return len(p), nil
	}
	return w.base.Write(p)
}

func init() {
	log.SetOutput(authzFilterWriter{base: os.Stderr})
}
