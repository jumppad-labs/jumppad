package events

import (
	"github.com/jumppad-labs/hclconfig/types"
)

type LifecycleEvent string

const (
	// LifecycleEventParsing is sent once the resource is about to be parsed
	LifecycleEventParsing LifecycleEvent = "parsing"

	// LifecycleEventParsed is sent once the resource has been successfully parsed
	LifecycleEventParsed LifecycleEvent = "parsed"

	// LifecycleEventParsingFailed is sent once the resource has failed to parse
	LifecycleEventParsingFailed LifecycleEvent = "parsing_failed"

	// LifecycleEventCreating is sent once the resource is about to be created
	LifecycleEventCreating LifecycleEvent = "creating"

	// LifecycleEventCreated is sent once the resource has been successfully created
	LifecycleEventCreated LifecycleEvent = "created"

	// LifecycleEventCreatingFailed is sent once the resource has failed to create
	LifecycleEventCreatingFailed LifecycleEvent = "creating_failed"

	// LifecycleEventDestroying is sent once the resource is about to be destroyed
	LifecycleEventDestroying LifecycleEvent = "destroying"

	// LifecycleEventDestroyed is sent once the resource has been destroyed
	LifecycleEventDestroyed LifecycleEvent = "destroyed"

	// LifecycleEventDestroyingFailed is sent once the resource has failed to create
	LifecycleEventDestroyingFailed LifecycleEvent = "destroying_failed"
)

type ParsingEvent struct {
	Path          string
	Variables     map[string]string
	VariablesFile string
}

type ParsedEvent struct {
	Resource *types.Meta
}

type ParsingFailedEvent struct {
	Path          string
	Variables     map[string]string
	VariablesFile string
	Error         error
}

type CreatingEvent struct {
	Resource *types.Meta
}

type CreatedEvent struct {
	Resource *types.Meta
}

type CreatingFailedEvent struct {
	Resource *types.Meta
	Error    error
}

type DestroyingEvent struct {
	Resource *types.Meta
}

type DestroyedEvent struct {
	Resource *types.Meta
}

type DestroyingFailedEvent struct {
	Resource *types.Meta
	Error    error
}
