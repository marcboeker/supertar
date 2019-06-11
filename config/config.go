package config

import "github.com/marcboeker/supertar/crypto"

// Config holds all parameters for an archive.
type Config struct {
	Path        string
	Password    []byte
	Compression bool
	Crypto      *crypto.Crypto
	ChunkSize   int
}
