package docs

import "github.com/jumppad-labs/jumppad/pkg/config"

// register the types and provider
func init() {
	config.RegisterResource(TypeDocs, &Docs{}, &DocsProvider{})
	config.RegisterResource(TypeChapter, &Chapter{}, &ChapterProvider{})
	config.RegisterResource(TypeTask, &Task{}, &TaskProvider{})
}
