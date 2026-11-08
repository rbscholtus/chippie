package chip8

// TraceEntry is one executed instruction, recorded before it ran.
type TraceEntry struct {
	PC    uint16
	Op    uint16
	I     uint16
	Frame uint32
}

// TraceBuffer is a ring buffer holding the most recent instructions. It is
// allocated once by [Machine.EnableTrace]; recording only stores raw values.
type TraceBuffer struct {
	entries []TraceEntry
	next    int  // where the next entry goes
	full    bool // the buffer has wrapped at least once
}

// Len returns the number of entries recorded, up to the buffer's capacity.
func (t *TraceBuffer) Len() int {
	if t.full {
		return len(t.entries)
	}

	return t.next
}

// Entries returns the recorded entries, oldest first, in a new slice.
func (t *TraceBuffer) Entries() []TraceEntry {
	out := make([]TraceEntry, 0, t.Len())
	if t.full {
		out = append(out, t.entries[t.next:]...)
	}

	return append(out, t.entries[:t.next]...)
}

func (t *TraceBuffer) record(e TraceEntry) {
	t.entries[t.next] = e

	t.next++
	if t.next == len(t.entries) {
		t.next = 0
		t.full = true
	}
}

func (t *TraceBuffer) clear() {
	t.next = 0
	t.full = false
}

// EnableTrace starts recording the last n instructions, allocating the
// buffer once. n <= 0 switches tracing off.
func (m *Machine) EnableTrace(n int) {
	if n <= 0 {
		m.trace = nil

		return
	}

	m.trace = &TraceBuffer{entries: make([]TraceEntry, n)}
}

// Trace returns the trace buffer, or nil when tracing is off.
func (m *Machine) Trace() *TraceBuffer { return m.trace }
