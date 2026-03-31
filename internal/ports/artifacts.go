package ports

type ArtifactStore interface {
	Put(relativePath string, data []byte) (string, error)
	ReadPreview(path string, maxBytes int) (string, bool, error)
	ResolveDisplayPath(path string) (string, error)
}
