package minecraft

import (
	"bytes"
	"io"
	"log/slog"
	"net"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestInternalPacketObserverReportsInternallyConsumedPacketAfterParse(t *testing.T) {
	var observed []InternalPacket
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	conn := newConn(client, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), DefaultProtocol, 0, false)
	conn.pool = DefaultProtocol.Packets(false)
	conn.expect(packet.IDChunkRadiusUpdated)
	conn.internalPacketFunc = func(event InternalPacket) {
		observed = append(observed, event)
	}

	// Serialise a real packet, parse it, then pass it through the expected-ID
	// dispatch. ChunkRadiusUpdated is consumed by Conn.handlePacket rather than
	// returned to ReadPacket, which is exactly the observability gap this hook
	// closes.
	var wire bytes.Buffer
	if err := (&packet.Header{PacketID: packet.IDChunkRadiusUpdated}).Write(&wire); err != nil {
		t.Fatal(err)
	}
	(&packet.ChunkRadiusUpdated{ChunkRadius: 1}).Marshal(DefaultProtocol.NewWriter(&wire, 0))
	data, err := parseData(wire.Bytes(), conn)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.handle(data); err != nil {
		t.Fatal(err)
	}
	if len(observed) != 1 {
		t.Fatalf("observed %d packets, want 1", len(observed))
	}
	if observed[0] != (InternalPacket{Sequence: 1, Name: "ChunkRadiusUpdated"}) {
		t.Fatalf("unexpected metadata-only observations: %#v", observed)
	}
}
