package patch

import "encoding/json"

type Patchable[T any] struct {
	Value  *T
	Sent   bool
	IsNull bool
}

func (p *Patchable[T]) UnmarshalJSON(data []byte) error {
	p.Sent = true
	if len(data) == 4 && string(data) == "null" {
		p.IsNull = true
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	p.Value = &v
	return nil
}

func (p *Patchable[T]) Set() bool {
	return p != nil && p.Sent
}

func (p *Patchable[T]) ValueOr(existing *T) *T {
	if !p.Set() {
		return existing
	}
	if p.IsNull {
		return nil
	}
	return p.Value
}
