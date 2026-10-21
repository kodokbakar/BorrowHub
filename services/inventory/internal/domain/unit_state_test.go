package domain

import (
	"errors"
	"testing"
)

func TestUnitStateMachine(t *testing.T) {
	validTransitions := []struct {
		from UnitStatus
		to   UnitStatus
	}{
		{StatusAvailable, StatusReserved},
		{StatusReserved, StatusBorrowed},
		{StatusBorrowed, StatusAvailable},
		{StatusAvailable, StatusMaintenance},
		{StatusMaintenance, StatusAvailable},
		{StatusAvailable, StatusRetired},
		{StatusMaintenance, StatusRetired},
	}

	for _, tt := range validTransitions {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			if !CanTransition(tt.from, tt.to) {
				t.Fatalf("expected valid transition from %s to %s", tt.from, tt.to)
			}
			if err := TransitionUnit(tt.from, tt.to); err != nil {
				t.Fatalf("unexpected error for %s to %s: %v", tt.from, tt.to, err)
			}
		})
	}

	invalidTransitions := []struct {
		from UnitStatus
		to   UnitStatus
	}{
		{StatusRetired, StatusBorrowed},
		{StatusRetired, StatusAvailable},
		{StatusBorrowed, StatusMaintenance},
		{StatusBorrowed, StatusRetired},
		{StatusMaintenance, StatusBorrowed},
		{StatusReserved, StatusMaintenance},
		{StatusAvailable, StatusBorrowed},
		{StatusAvailable, StatusAvailable},
		{StatusReserved, StatusReserved},
		{StatusBorrowed, StatusBorrowed},
		{StatusMaintenance, StatusMaintenance},
		{StatusRetired, StatusRetired},
	}

	for _, tt := range invalidTransitions {
		t.Run("invalid_"+string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			if CanTransition(tt.from, tt.to) {
				t.Fatalf("expected invalid transition from %s to %s", tt.from, tt.to)
			}
			err := TransitionUnit(tt.from, tt.to)
			if !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf("expected ErrInvalidTransition, got %v", err)
			}
		})
	}
}

func TestUnitGuards(t *testing.T) {
	unit := &Unit{Status: StatusAvailable}
	item := &Item{IsLendable: true}

	if !unit.CanLend(item) {
		t.Fatal("expected unit to be lendable when available and item is lendable")
	}

	item.IsLendable = false
	if unit.CanLend(item) {
		t.Fatal("expected unit not lendable when item is not lendable")
	}

	item.IsLendable = true
	unit.Status = StatusReserved
	if unit.CanLend(item) {
		t.Fatal("expected reserved unit not lendable")
	}

	unit.Status = StatusRetired
	if unit.IsActive() {
		t.Fatal("expected retired unit to not be active")
	}

	unit.Status = StatusAvailable
	if !unit.IsActive() {
		t.Fatal("expected available unit to be active")
	}
}

func TestAdminAndLendingTransitions(t *testing.T) {
	// Admin transitions
	if !CanAdminTransition(StatusAvailable, StatusMaintenance) {
		t.Fatal("admin should be able to start maintenance")
	}
	if !CanAdminTransition(StatusMaintenance, StatusAvailable) {
		t.Fatal("admin should be able to complete maintenance")
	}
	if !CanAdminTransition(StatusAvailable, StatusRetired) {
		t.Fatal("admin should be able to retire available unit")
	}
	if !CanAdminTransition(StatusMaintenance, StatusRetired) {
		t.Fatal("admin should be able to retire maintenance unit")
	}
	if CanAdminTransition(StatusAvailable, StatusBorrowed) {
		t.Fatal("admin should not transition directly to borrowed")
	}
	if CanAdminTransition(StatusAvailable, StatusReserved) {
		t.Fatal("admin should not transition directly to reserved")
	}

	// Lending transitions
	if !CanLendingTransition(StatusAvailable, StatusReserved) {
		t.Fatal("lending should be able to reserve")
	}
	if !CanLendingTransition(StatusReserved, StatusBorrowed) {
		t.Fatal("lending should be able to checkout")
	}
	if !CanLendingTransition(StatusBorrowed, StatusAvailable) {
		t.Fatal("lending should be able to return")
	}
	if !CanLendingTransition(StatusReserved, StatusAvailable) {
		t.Fatal("lending should be able to cancel reservation")
	}
	if CanLendingTransition(StatusAvailable, StatusBorrowed) {
		t.Fatal("lending cannot directly checkout without reservation")
	}
	if CanLendingTransition(StatusAvailable, StatusMaintenance) {
		t.Fatal("lending cannot trigger maintenance directly")
	}
}
