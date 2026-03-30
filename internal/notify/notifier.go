package notify

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/tristanj/dusty/internal/haptic"
	"github.com/tristanj/dusty/internal/mesh"
	"github.com/tristanj/dusty/internal/state"
)

// messageStorer is the subset of mesh.MessageStore needed by Notifier.
type messageStorer interface {
	StoreReceived(msg mesh.MeshMessage)
}

// NotificationPayload is the payload for EventNotificationTriggered.
type NotificationPayload struct {
	Message mesh.MeshMessage
	IsIdle  bool // True if notification was suppressed due to user inactivity.
}

// Notifier orchestrates sound, haptic, and event bus notifications for
// incoming mesh messages. It also tracks user activity to suppress overlays
// when the user has been idle for longer than IdleTimeout.
type Notifier struct {
	bus         *state.EventBus
	sound       SoundPlayer
	haptic      haptic.HapticProvider
	store       messageStorer
	soundPath   string
	IdleTimeout time.Duration

	lastActivity atomic.Int64 // Unix nanoseconds of last recorded activity.
}

// NotifierConfig holds configuration for the Notifier.
type NotifierConfig struct {
	SoundPath   string
	IdleTimeout time.Duration // Default 5 minutes.
}

// NewNotifier creates a Notifier. store may be nil (messages are not persisted).
func NewNotifier(cfg NotifierConfig, bus *state.EventBus, sound SoundPlayer, hap haptic.HapticProvider, store messageStorer) *Notifier {
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 5 * time.Minute
	}
	n := &Notifier{
		bus:         bus,
		sound:       sound,
		haptic:      hap,
		store:       store,
		soundPath:   cfg.SoundPath,
		IdleTimeout: cfg.IdleTimeout,
	}
	n.RecordActivity() // Start as active.
	return n
}

// RecordActivity updates the last-activity timestamp. Call on any user input.
func (n *Notifier) RecordActivity() {
	n.lastActivity.Store(time.Now().UnixNano())
}

// IsUserIdle returns true if no activity has been recorded within IdleTimeout.
func (n *Notifier) IsUserIdle() bool {
	last := time.Unix(0, n.lastActivity.Load())
	return time.Since(last) > n.IdleTimeout
}

// Notify processes an incoming mesh message: plays sound, triggers haptic,
// publishes EventNotificationTriggered. If the user is idle, the overlay is
// suppressed and the message is stored as unread silently.
func (n *Notifier) Notify(msg mesh.MeshMessage) {
	idle := n.IsUserIdle()

	if !idle {
		// Play notification sound.
		if n.sound != nil {
			if err := n.sound.PlayFile(n.soundPath); err != nil {
				log.Printf("[notify] sound error: %v", err)
			}
		}
		// Trigger haptic.
		if n.haptic != nil {
			if err := n.haptic.Vibrate(haptic.PatternDouble); err != nil {
				log.Printf("[notify] haptic error: %v", err)
			}
		}
	}

	// Always publish the event — TUI/REPL decide whether to show the overlay
	// based on the IsIdle flag.
	n.bus.Publish(state.NewEvent(state.EventNotificationTriggered, NotificationPayload{
		Message: msg,
		IsIdle:  idle,
	}))
}

// Start subscribes to EventMeshMessageReceived and begins dispatching notifications.
func (n *Notifier) Start(stopCh <-chan struct{}) {
	ch := n.bus.Subscribe(state.EventMeshMessageReceived)
	go func() {
		for {
			select {
			case evt, ok := <-ch:
				if !ok {
					return
				}
				if msg, ok := evt.Payload.(mesh.MeshMessage); ok {
					// Persist to inbox before notifying.
					if n.store != nil {
						n.store.StoreReceived(msg)
					}
					n.Notify(msg)
				}
			case <-stopCh:
				return
			}
		}
	}()
}
