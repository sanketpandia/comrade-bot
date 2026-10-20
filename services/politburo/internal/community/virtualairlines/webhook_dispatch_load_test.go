package virtualairlines

import "testing"

func TestPreferredServersFromBasic(t *testing.T) {
	servers, err := preferredServersFromBasic([]byte(`{"preferred_servers":["Expert","casual","casual"]}`))
	if err != nil {
		t.Fatalf("preferredServersFromBasic() = %v", err)
	}
	if len(servers) != 2 || servers[0] != "casual" || servers[1] != "expert" {
		t.Fatalf("servers = %#v", servers)
	}

	servers, err = preferredServersFromBasic([]byte(`{}`))
	if err != nil || len(servers) != 0 {
		t.Fatalf("empty config servers = %#v err = %v", servers, err)
	}
	if _, err := preferredServersFromBasic([]byte(`{`)); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}
