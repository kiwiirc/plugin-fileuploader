package events

import (
	"sync"

	"github.com/tus/tusd"
	"github.com/tus/tusd/cmd/tusd/cli/hooks"
)

// how many events can be unread by a listener before everything starts to block
const bufferSize = 16

type TusEvent struct {
	Info tusd.FileInfo
	Type hooks.HookType
}

type TusEventBroadcaster struct {
	mu        sync.RWMutex
	listeners []chan *TusEvent
	quitChan  chan struct{} // closes to signal quitting
}

func NewTusEventBroadcaster(handler *tusd.UnroutedHandler) *TusEventBroadcaster {
	broadcaster := &TusEventBroadcaster{
		quitChan: make(chan struct{}),
	}

	go broadcaster.readLoop(handler)

	return broadcaster
}

func (b *TusEventBroadcaster) Listen() <-chan *TusEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	newListener := make(chan *TusEvent, bufferSize)

	b.listeners = append(b.listeners, newListener)

	return newListener
}

func (b *TusEventBroadcaster) Unlisten(listener chan *TusEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// delete the listener
	kept := 0
	for _, l := range b.listeners {
		if l == listener {
			b.listeners[kept] = listener
			kept++
		}
	}
	b.listeners = b.listeners[:kept]
}

func (b *TusEventBroadcaster) readLoop(handler *tusd.UnroutedHandler) {
	for {
		select {
		case info := <-handler.CompleteUploads:
			b.broadcast(hooks.HookPostFinish, info)
		case info := <-handler.TerminatedUploads:
			b.broadcast(hooks.HookPostTerminate, info)
		case info := <-handler.UploadProgress:
			b.broadcast(hooks.HookPostReceive, info)
		case info := <-handler.CreatedUploads:
			b.broadcast(hooks.HookPostCreate, info)
		case _, ok := <-b.quitChan:
			if !ok {
				return
			}
		}
	}
}

func (b *TusEventBroadcaster) broadcast(hookType hooks.HookType, info tusd.FileInfo) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	event := &TusEvent{
		Type: hookType,
		Info: info,
	}

	for _, l := range b.listeners {
		l <- event
	}
}

func (b *TusEventBroadcaster) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, l := range b.listeners {
		close(l)
	}

	close(b.quitChan)
}
