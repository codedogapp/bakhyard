package push_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/databasetest"
	"github.com/codedogapp/bakhyard/services/notify/internal/migrations"
	"github.com/codedogapp/bakhyard/services/notify/internal/push"
	"github.com/codedogapp/bakhyard/services/notify/internal/subscription"
	"github.com/codedogapp/bakhyard/services/notify/internal/vapid"
)

// pushed is a push that the fake push service got.
type pushed struct {
	path   string
	header http.Header
	body   []byte
}

// pushService is a fake push service, over TLS. It answers a push to /push/<status>/<name> with
// that status, and keeps each push it got.
type pushService struct {
	server *httptest.Server
	mu     sync.Mutex
	got    []pushed
}

func newPushService(t *testing.T) *pushService {
	t.Helper()

	service := &pushService{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /push/{status}/{name}", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		service.mu.Lock()
		service.got = append(service.got, pushed{r.URL.Path, r.Header, body})
		service.mu.Unlock()

		status := 201
		if r.PathValue("status") == "410" {
			status = http.StatusGone
		} else if r.PathValue("status") == "500" {
			status = http.StatusInternalServerError
		}

		w.WriteHeader(status)
	})
	service.server = httptest.NewTLSServer(mux)
	t.Cleanup(service.server.Close)

	return service
}

func (p *pushService) endpoint(status, name string) string {
	return p.server.URL + "/push/" + status + "/" + name
}

func (p *pushService) pushes() []pushed {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]pushed(nil), p.got...)
}

func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	schema := databasetest.NewSchema(t)

	pool, err := database.Open(t.Context(), schema.URL, schema.Password, migrations.FS())
	noError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

// newSender returns a sender whose devices are subs, and what it logs.
func newSender(
	t *testing.T, service *pushService, subs ...subscription.Subscription,
) (*push.Sender, *subscription.Store, *bytes.Buffer) {
	t.Helper()

	store := subscription.NewStore(newPool(t))
	for _, sub := range subs {
		noError(t, store.Add(t.Context(), sub))
	}

	private, err := vapid.Generate()
	noError(t, err)

	key, err := vapid.Parse(private)
	noError(t, err)

	logs := &bytes.Buffer{}
	sender := &push.Sender{
		Key:     key,
		Devices: store,
		Client:  service.server.Client(),
		Logger:  slog.New(slog.NewTextHandler(logs, nil)),
	}

	return sender, store, logs
}

func TestSendPushesTheMessageToEachDevice(t *testing.T) {
	service := newPushService(t)
	devices := []device{newDevice(t), newDevice(t)}
	sender, _, logs := newSender(t, service,
		devices[0].at(service.endpoint("201", "first")),
		devices[1].at(service.endpoint("201", "second")),
	)

	err := sender.Send(t.Context(), push.Message{Title: "atlas", Body: "Merge it?", URL: "/ask"})
	noError(t, err)

	got := service.pushes()
	if len(got) != len(devices) {
		t.Fatalf("the push service got %d pushes, want %d", len(got), len(devices))
	}

	want := `{"title":"atlas","body":"Merge it?","url":"/ask"}`
	for number, device := range devices {
		if message := device.decrypt(t, got[number].body); message != want {
			t.Errorf("push %d = %s, want %s", number, message, want)
		}

		header := got[number].header
		if !strings.HasPrefix(header.Get("Authorization"), "vapid t=") ||
			header.Get("Ttl") != "86400" || header.Get("Urgency") != "high" {
			t.Errorf("push %d has the headers %v", number, header)
		}
	}

	if logs.Len() != 0 {
		t.Errorf("logged %s", logs)
	}
}

func TestSendTitlesAMessageWithoutOneAndCutsALongBody(t *testing.T) {
	service := newPushService(t)
	phone := newDevice(t)
	sender, _, _ := newSender(t, service, phone.at(service.endpoint("201", "phone")))

	noError(t, sender.Send(t.Context(), push.Message{Body: strings.Repeat("é", 600)}))

	want := `{"title":"Bakhyard","body":"` + strings.Repeat("é", 499) + `…"}`
	if got := phone.decrypt(t, service.pushes()[0].body); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestSendRefusesAUrlThatIsNotAPathInTheApp(t *testing.T) {
	service := newPushService(t)
	sender, _, _ := newSender(t, service)

	for _, url := range []string{"https://evil.example/", "//evil.example", "ask"} {
		if err := sender.Send(t.Context(), push.Message{Body: "x", URL: url}); err == nil {
			t.Errorf("Send accepted the url %q", url)
		}
	}
}

func TestSendForgetsADeviceWhoseSubscriptionIsGoneAndKeepsPushingToTheRest(t *testing.T) {
	service := newPushService(t)
	gone, kept := newDevice(t), newDevice(t)
	sender, store, logs := newSender(t, service,
		gone.at(service.endpoint("410", "gone")), kept.at(service.endpoint("201", "kept")))

	noError(t, sender.Send(t.Context(), push.Message{Body: "hi"}))

	left, err := store.List(t.Context())
	if err != nil || len(left) != 1 || left[0].Endpoint != service.endpoint("201", "kept") {
		t.Errorf("devices left: %+v (%v)", left, err)
	}

	if len(service.pushes()) != 2 || !strings.Contains(logs.String(), "forgot a device") {
		t.Errorf("pushes: %d, log: %s", len(service.pushes()), logs)
	}
}

func TestSendLogsARefusalAndAnUnreachableServiceWithoutTheEndpoint(t *testing.T) {
	service := newPushService(t)
	refusing := newDevice(t).at(service.endpoint("500", "refused"))
	unreachable := newDevice(t).at("https://127.0.0.1:1/push/secret-endpoint")
	sender, _, logs := newSender(t, service, refusing, unreachable)

	noError(t, sender.Send(t.Context(), push.Message{Body: "hi"}))

	got := logs.String()
	if !strings.Contains(got, "push refused") || !strings.Contains(got, "push failed") ||
		strings.Contains(got, "secret-endpoint") {
		t.Errorf("log: %s", got)
	}
}

func TestSendFailsWithoutTheDatabase(t *testing.T) {
	service := newPushService(t)
	sender, _, _ := newSender(t, service)

	pool := newPool(t)
	pool.Close()

	sender.Devices = subscription.NewStore(pool)

	if err := sender.Send(t.Context(), push.Message{Body: "x"}); err == nil {
		t.Error("Send succeeded")
	}
}
