package android

import "testing"

func TestPartitionedDatrPool_RecyclesAfterUsageLimit(t *testing.T) {
	p := NewPartitionedPool(1)
	var removed []string
	p.SetRemoveHook(func(datr string) { removed = append(removed, datr) })
	p.Register(1)
	p.AddDatrRaw("d1")

	if got := p.GetNext(1); got != "d1" {
		t.Fatalf("GetNext = %q, want d1", got)
	}
	p.IncrementUsage("d1")
	if p.Size() != 0 {
		t.Fatalf("Size = %d, want 0 while datr is exhausted", p.Size())
	}
	if len(removed) != 0 {
		t.Fatalf("removed = %#v, want no remove hook for usage limit", removed)
	}
	if got := p.GetNext(1); got != "d1" {
		t.Fatalf("GetNext after recycle = %q, want d1", got)
	}
	_, _, _, usage := p.GetStats("d1")
	if usage != 0 {
		t.Fatalf("usage after recycle = %d, want 0", usage)
	}
}

func TestPartitionedDatrPool_RemovesAtCheckpointLimit(t *testing.T) {
	p := NewPartitionedPool(9)
	p.SetMaxCheckpoint(2)
	p.Register(1)
	p.AddDatrRaw("d1")

	p.RecordResult("d1", "checkpoint")
	if p.Size() != 1 {
		t.Fatalf("Size after first checkpoint = %d, want 1", p.Size())
	}
	p.RecordResult("d1", "checkpoint")
	if p.Size() != 0 {
		t.Fatalf("Size after checkpoint limit = %d, want 0", p.Size())
	}
}
