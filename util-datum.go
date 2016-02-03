package dht

// An arbitrary non-peer data item stored in the DHT.
type Datum struct {
	Value string // The datum value.

	Key        krPublicKey // If set, this is a mutable datum.
	Salt       []byte      // Salt. Only used for mutable data.
	Signature  krSignature // Value signature. Only used for mutable data.
	SequenceNo uint64      // Sequence number. Only used for mutable data.
}

func (d *Datum) IsMutable() bool {
	return d.Key.IsWellFormed()
}
