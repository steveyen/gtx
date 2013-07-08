package cbtx

type Timestamp uint64
type Addr string
type Key string

type Write struct {
	Key  Key
	Val  []byte    // When nil, the write is a deletion.
	Ts   Timestamp // Writes are ordered.
	Sibs []Key
}

type Server interface {
	Get(k Key, tsRequired Timestamp) (*Write, error)
	Set(w Write) error
}

type ServerController struct {
	ss       ServerStore
	replicas []Addr
}

type ServerStore interface {
	GoodFind(k Key, tsMininum Timestamp) (*Write, error)
	PendingGet(k Key, tsRequired Timestamp) (*Write, error)
	PendingAdd(w Write) error
	AcksIncr(fromReplica Addr, ts Timestamp) (int, error)
	Promote(ts Timestamp) error
}

func NewServerController(ss ServerStore, replicas []Addr) *ServerController {
	return &ServerController{ss, replicas}
}

func (s *ServerController) Set(w Write) error {
	err := s.ss.PendingAdd(w)
	if err != nil {
		return err
	}
	for _, k := range w.Sibs {
		for _, replica := range s.ReplicasFor(k) {
			s.SendNotify(replica, w.Ts)
		}
	}
	// TODO: Asynchronously send w to other replicas via anti-entropy.
	return nil
}

func (s *ServerController) Get(k Key, tsRequired Timestamp) (*Write, error) {
	w, err := s.ss.GoodFind(k, tsRequired)
	if err != nil || w != nil {
		return w, err
	}
	if tsRequired == 0 {
		return nil, nil
	}
	return s.ss.PendingGet(k, tsRequired)
}

func (s *ServerController) ReceiveNotify(fromReplica Addr, ts Timestamp) error {
	acks, err := s.ss.AcksIncr(fromReplica, ts)
	if err != nil {
		return err
	}
	if acks >= s.AcksNeeded(ts) {
		return s.ss.Promote(ts)
	}
	return nil
}

func (s *ServerController) SendNotify(toReplica Addr, ts Timestamp) error {
	return nil
}

func (s *ServerController) ReplicasFor(k Key) []Addr {
	return s.replicas
}

func (s *ServerController) AcksNeeded(ts Timestamp) int {
	return len(s.replicas)
}
