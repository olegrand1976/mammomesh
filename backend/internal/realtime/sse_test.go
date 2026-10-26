package realtime

import (
	"bytes"
	"net/http/httptest"
	"testing"
)

func TestWriteMessage(t *testing.T) {
	var b bytes.Buffer
	WriteMessage(&b, Message{ID: 42, Type: "nouveau_meilleur", Data: []byte(`{"a":1}`)})
	want := "id: 42\nevent: nouveau_meilleur\ndata: {\"a\":1}\n\n"
	if b.String() != want {
		t.Fatalf("got %q want %q", b.String(), want)
	}
	b.Reset()
	WriteMessage(&b, Message{Type: "progression", Data: []byte("l1\nl2")})
	if b.String() != "event: progression\ndata: l1\ndata: l2\n\n" {
		t.Fatalf("message sans identifiant mal formé : %q", b.String())
	}
}

func TestLastEventID(t *testing.T) {
	r := httptest.NewRequest("GET", "/x?lastEventId=7", nil)
	if LastEventID(r) != 7 {
		t.Fatal("paramètre lastEventId ignoré")
	}
	r.Header.Set("Last-Event-ID", "12")
	if LastEventID(r) != 12 {
		t.Fatal("en-tête Last-Event-ID ignoré")
	}
	r.Header.Set("Last-Event-ID", "-3")
	if LastEventID(r) != 0 {
		t.Fatal("identifiant négatif accepté")
	}
}
