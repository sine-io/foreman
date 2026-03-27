package ports

type ArtifactStore interface {
	Put(relativePath string, data []byte) (string, error)
}
