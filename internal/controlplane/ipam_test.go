package controlplane

import "testing"

func TestIPAMAllocatesDeterministicUsableIPv4Addresses(t *testing.T) {
	ipam, err := NewIPAM("10.44.0.0/29")
	if err != nil {
		t.Fatalf("new ipam: %v", err)
	}

	first, err := ipam.Allocate("agent-a")
	if err != nil {
		t.Fatalf("allocate first: %v", err)
	}
	second, err := ipam.Allocate("agent-b")
	if err != nil {
		t.Fatalf("allocate second: %v", err)
	}

	if first != "10.44.0.1" {
		t.Fatalf("first address = %q, want 10.44.0.1", first)
	}
	if second != "10.44.0.2" {
		t.Fatalf("second address = %q, want 10.44.0.2", second)
	}
	if first == second {
		t.Fatalf("expected unique addresses, got %q twice", first)
	}
}

func TestIPAMReturnsStableLeaseForExistingAgent(t *testing.T) {
	ipam, err := NewIPAM("10.44.0.0/29")
	if err != nil {
		t.Fatalf("new ipam: %v", err)
	}

	first, err := ipam.Allocate("agent-a")
	if err != nil {
		t.Fatalf("allocate first: %v", err)
	}
	second, err := ipam.Allocate("agent-a")
	if err != nil {
		t.Fatalf("allocate second: %v", err)
	}

	if first != second {
		t.Fatalf("stable lease = %q then %q", first, second)
	}
}

func TestIPAMReportsExhaustion(t *testing.T) {
	ipam, err := NewIPAM("10.44.0.0/30")
	if err != nil {
		t.Fatalf("new ipam: %v", err)
	}

	if _, err := ipam.Allocate("agent-a"); err != nil {
		t.Fatalf("allocate agent-a: %v", err)
	}
	if _, err := ipam.Allocate("agent-b"); err != nil {
		t.Fatalf("allocate agent-b: %v", err)
	}
	if _, err := ipam.Allocate("agent-c"); err == nil {
		t.Fatal("expected exhaustion error")
	}
}

func TestIPAMRejectsInvalidCIDR(t *testing.T) {
	if _, err := NewIPAM("not-a-cidr"); err == nil {
		t.Fatal("expected invalid cidr error")
	}
}
