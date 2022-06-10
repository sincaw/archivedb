package pkg

type ChunkKey = []byte

type Meta struct {
	// inputs
	// mine type: https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types
	Mime string `json:"mime"`
	// chunk size in bytes, value will split to chunk when it is set
	ChunkSize int `json:"chunkSize"`

	// outputs
	// value len in bytes, it will automatically set
	TotalLen int `json:"totalLen"`
	// chunk list keys
	Chunks []ChunkKey `json:"chunks"`
}
