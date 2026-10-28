package change_test

import (
	"testing"

	"github.com/codedogapp/bakhyard/services/nudge/internal/change"
)

func TestPublishSignalsEverySubscriberOnceAndMergesRepeats(t *testing.T) {
	var hub change.Hub

	first, endFirst := hub.Subscribe()
	second, endSecond := hub.Subscribe()

	defer endSecond()

	hub.Publish()
	hub.Publish()

	for name, watcher := range map[string]<-chan struct{}{"first": first, "second": second} {
		<-watcher

		select {
		case <-watcher:
			t.Errorf("%s got a second signal", name)
		default:
		}
	}

	endFirst()
	hub.Publish()

	select {
	case <-first:
		t.Error("an ended subscription got a signal")
	default:
	}

	<-second
}
