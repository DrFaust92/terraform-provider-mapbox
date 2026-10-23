// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"scopes are invalid"}`))
	}))
	defer server.Close()

	token := "sk.secret"
	client := &Client{AccessToken: &token, BaseURL: server.URL + "/"}

	_, err := client.Get("tokens/v2/user")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "scopes are invalid") {
		t.Errorf("expected API message in error, got %q", err)
	}
}

func TestClientRedactsAccessTokenOnTransportError(t *testing.T) {
	token := "sk.super+secret/token"
	// Nothing listens on port 1, so the request fails at the transport layer.
	client := &Client{AccessToken: &token, BaseURL: "http://127.0.0.1:1/"}

	_, err := client.Get("tokens/v2/user")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "secret") {
		t.Errorf("access token leaked in error: %q", err)
	}
	if !strings.Contains(err.Error(), "REDACTED") {
		t.Errorf("expected redacted URL in error, got %q", err)
	}
}

func TestListTokensPagination(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("access_token") != "pk.test" {
			t.Errorf("missing access token on %s", r.URL)
		}

		switch r.URL.Query().Get("start") {
		case "":
			w.Header().Set("Link", fmt.Sprintf(`<%s/tokens/v2/user?limit=100&start=cursor1>; rel="next"`, server.URL))
			_, _ = w.Write([]byte(`[{"id":"a","note":"a","scopes":["styles:read"]}]`))
		case "cursor1":
			_, _ = w.Write([]byte(`[{"id":"b","note":"b","scopes":["styles:read"],"usage":"sk"}]`))
		default:
			t.Errorf("unexpected start %q", r.URL.Query().Get("start"))
		}
	}))
	defer server.Close()

	token := "pk.test"
	client := &Client{AccessToken: &token, BaseURL: server.URL + "/"}

	tokens, err := client.listTokens("user")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 2 || *tokens[0].Id != "a" || *tokens[1].Id != "b" {
		t.Fatalf("unexpected tokens: %+v", tokens)
	}
}

func TestListTokensNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer server.Close()

	token := "pk.test"
	client := &Client{AccessToken: &token, BaseURL: server.URL + "/"}

	_, err := client.listTokens("user")
	if !isNotFound(err) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestNextPageStart(t *testing.T) {
	cases := map[string]string{
		"": "",
		`<https://api.mapbox.com/tokens/v2/u?limit=100&start=abc>; rel="next"`:                                               "abc",
		`<https://api.mapbox.com/tokens/v2/u?start=x>; rel="prev", <https://api.mapbox.com/tokens/v2/u?start=y>; rel="next"`: "y",
		`<https://api.mapbox.com/tokens/v2/u?start=x>; rel=next`:                                                             "x",
		`<https://api.mapbox.com/tokens/v2/u?start=x>; rel="last"`:                                                           "",
	}

	for header, want := range cases {
		if got := nextPageStart(header); got != want {
			t.Errorf("nextPageStart(%q) = %q, want %q", header, got, want)
		}
	}
}

func TestTokenId(t *testing.T) {
	for _, id := range []string{"", "abc", ":user", "abc:", "a:b:c"} {
		if _, _, err := tokenId(id); err == nil {
			t.Errorf("expected error for %q", id)
		}
	}

	id, user, err := tokenId("abc:user")
	if err != nil || id != "abc" || user != "user" {
		t.Errorf("unexpected result: %q %q %v", id, user, err)
	}
}
