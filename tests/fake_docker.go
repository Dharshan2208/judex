package tests

import (
	"encoding/binary"
	"encoding/json"
	"net/url"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/docker/docker/client"
)

type ExecResponse struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type FakeDocker struct {
	t *testing.T

	mu            sync.Mutex
	nextExecID    int
	Commands      [][]string
	ExecResponses map[string]ExecResponse
	FailStartExec map[string]bool

	server *httptest.Server
}

func NewFakeDocker(t *testing.T) *FakeDocker {
	t.Helper()
	fd := &FakeDocker{
		t:             t,
		ExecResponses: make(map[string]ExecResponse),
		FailStartExec: make(map[string]bool),
	}

	fd.server = httptest.NewServer(http.HandlerFunc(fd.handle))
	t.Cleanup(fd.server.Close)
	return fd
}

func (f *FakeDocker) NewClient(t *testing.T) *client.Client {
	t.Helper()
	u, err := url.Parse(f.server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	cli, err := client.NewClientWithOpts(
		client.WithHost("http://"+u.Host),
		client.WithVersion("1.44"),
		client.WithHTTPClient(f.server.Client()),
	)
	if err != nil {
		t.Fatalf("new docker client: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return cli
}

func (f *FakeDocker) SetExecResponse(execID string, resp ExecResponse) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ExecResponses[execID] = resp
}

func (f *FakeDocker) handle(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	switch {
	case r.Method == http.MethodPost && strings.Contains(p, "/containers/create"):
		_ = json.NewEncoder(w).Encode(map[string]string{"Id": "container-1"})
		return
	case r.Method == http.MethodPost && strings.Contains(p, "/start"):
		w.WriteHeader(http.StatusNoContent)
		return
	case r.Method == http.MethodPut && strings.Contains(p, "/archive"):
		w.WriteHeader(http.StatusOK)
		return
	case r.Method == http.MethodDelete && strings.Contains(p, "/containers/"):
		w.WriteHeader(http.StatusNoContent)
		return
	case r.Method == http.MethodPost && strings.Contains(p, "/kill"):
		w.WriteHeader(http.StatusNoContent)
		return
	case r.Method == http.MethodPost && strings.Contains(p, "/containers/") && strings.HasSuffix(p, "/exec"):
		f.handleExecCreate(w, r)
		return
	case r.Method == http.MethodPost && strings.Contains(p, "/exec/") && strings.Contains(p, "/start"):
		f.handleExecStart(w, r)
		return
	case r.Method == http.MethodGet && strings.Contains(p, "/exec/") && strings.Contains(p, "/json"):
		f.handleExecInspect(w, r)
		return
	default:
		http.Error(w, "not implemented: "+r.Method+" "+p, http.StatusNotFound)
	}
}

func (f *FakeDocker) handleExecCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Cmd []string `json:"Cmd"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	f.mu.Lock()
	defer f.mu.Unlock()

	f.Commands = append(f.Commands, body.Cmd)
	f.nextExecID++
	execID := "exec-" + strconv.Itoa(f.nextExecID)
	if _, ok := f.ExecResponses[execID]; !ok {
		f.ExecResponses[execID] = ExecResponse{ExitCode: 0}
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"Id": execID})
}

func (f *FakeDocker) handleExecStart(w http.ResponseWriter, r *http.Request) {
	execID := pathSegmentBetween(r.URL.Path, "/exec/", "/start")

	f.mu.Lock()
	resp := f.ExecResponses[execID]
	fail := f.FailStartExec[execID]
	f.mu.Unlock()
	if fail {
		http.Error(w, "exec start failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(muxStream(1, resp.Stdout))
	_, _ = w.Write(muxStream(2, resp.Stderr))
}

func (f *FakeDocker) handleExecInspect(w http.ResponseWriter, r *http.Request) {
	execID := pathSegmentBetween(r.URL.Path, "/exec/", "/json")

	f.mu.Lock()
	resp := f.ExecResponses[execID]
	f.mu.Unlock()

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ID":       execID,
		"ExitCode": resp.ExitCode,
		"Running":  false,
	})
}

func muxStream(stream byte, payload string) []byte {
	if payload == "" {
		return nil
	}

	data := []byte(payload)
	out := make([]byte, 8+len(data))
	out[0] = stream
	binary.BigEndian.PutUint32(out[4:8], uint32(len(data)))
	copy(out[8:], data)
	return out
}

func pathSegmentBetween(path, start, end string) string {
	i := strings.Index(path, start)
	if i < 0 {
		return ""
	}
	sub := path[i+len(start):]
	j := strings.Index(sub, end)
	if j < 0 {
		return sub
	}
	return sub[:j]
}
