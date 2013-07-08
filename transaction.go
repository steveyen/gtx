package cbtx

type Transaction struct {
	s        Server
	ts       Timestamp
	writes   map[Key][]byte
	required map[Key]Timestamp
}

func Begin(s Server, ts Timestamp) *Transaction {
	return &Transaction{
		s:        s,
		ts:       ts, // Should be clientId + logicalClock.
		writes:   map[Key][]byte{},
		required: map[Key]Timestamp{},
	}
}

func (t *Transaction) Set(k Key, v []byte) {
	t.writes[k] = v
}

func (t *Transaction) Del(k Key) {
	t.writes[k] = nil
}

func (t *Transaction) Get(k Key) ([]byte, error) {
	v, ok := t.writes[k]
	if ok {
		return v, nil // For per-Transaction read-your-writes.
	}
	tsRequired, _ := t.required[k]
	w, err := t.s.Get(k, tsRequired)
	if err != nil || w == nil {
		return nil, err
	}
	for _, sibKey := range w.Sibs {
		sibTsRequired, _ := t.required[sibKey]
		if w.Ts > sibTsRequired {
			t.required[sibKey] = w.Ts
		}
	}
	return w.Val, nil
}

func (t *Transaction) Commit() error {
	sibs := make([]Key, 0, len(t.writes))
	for k, _ := range t.writes {
		sibs = append(sibs, k)
	}
	for k, v := range t.writes {
		err := t.s.Set(Write{Key: k, Val: v, Ts: t.ts, Sibs: sibs})
		if err != nil {
			return err
		}
	}
	t.done()
	return nil
}

func (t *Transaction) Abort() error {
	t.done()
	return nil
}

func (t *Transaction) done() {
	t.s = nil
	t.writes = nil
	t.required = nil
}
