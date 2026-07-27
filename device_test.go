package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"
)

func TestPollDeviceToken_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "authorization_pending", "error_description": "The user has not yet authorized the device"}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	
	initialGoroutines := runtime.NumGoroutine()

	errChan := make(chan error, 1)
	go func() {
		_, err := PollDeviceToken(ctx, server.Client(), server.URL, "client_id", "device_code", 10*time.Millisecond)
		errChan <- err
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errChan:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("test timed out: PollDeviceToken did not return after context cancellation")
	}

	time.Sleep(50 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	if finalGoroutines > initialGoroutines+2 {
		t.Errorf("possible goroutine leak: started with %d, ended with %d", initialGoroutines, finalGoroutines)
	}
}