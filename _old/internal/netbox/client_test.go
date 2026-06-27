package netbox

import (
	"fmt"
	"net/http"
	"testing"
)

func TestClient_TokenProtected(t *testing.T) {
	c := &Client{
		Client: &http.Client{},
	}

	if c.HasToken() {
		t.Error("HasToken should return false when token is empty")
	}

	token := "token"
	c.SetToken(token)

	if !c.HasToken() {
		t.Error("HasToken should return true when token is set")
	}

	p := fmt.Sprintf("%#v", c.token)
	if p == token {
		t.Errorf("Token should be redacted, got %s", p)
	}
}
