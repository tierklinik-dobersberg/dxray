package main

type contextKey struct {
	name string
}

var (
	// ContextKeyDXR is used for a DXR
	ContextKeyDXR = &contextKey{"dxr"}

	// ContextKeyIndexer is used for a StudyIndexer
	ContextKeyIndexer = &contextKey{"study-indexer"}
)
