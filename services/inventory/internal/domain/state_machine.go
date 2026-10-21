package domain

// CanTransition determines whether a unit status transition is globally valid according to the state machine.
func CanTransition(from, to UnitStatus) bool {
	if from == to {
		return false
	}

	switch from {
	case StatusAvailable:
		return to == StatusReserved || to == StatusMaintenance || to == StatusRetired
	case StatusReserved:
		return to == StatusBorrowed || to == StatusAvailable
	case StatusBorrowed:
		return to == StatusAvailable
	case StatusMaintenance:
		return to == StatusAvailable || to == StatusRetired
	case StatusRetired:
		return false
	default:
		return false
	}
}

// CanAdminTransition checks whether an administrative user can perform this status transition.
func CanAdminTransition(from, to UnitStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case StatusAvailable:
		return to == StatusMaintenance || to == StatusRetired
	case StatusMaintenance:
		return to == StatusAvailable || to == StatusRetired
	default:
		return false
	}
}

// CanLendingTransition checks whether lending operations can perform this status transition.
func CanLendingTransition(from, to UnitStatus) bool {
	if from == to {
		return false
	}
	switch from {
	case StatusAvailable:
		return to == StatusReserved
	case StatusReserved:
		return to == StatusBorrowed || to == StatusAvailable
	case StatusBorrowed:
		return to == StatusAvailable
	default:
		return false
	}
}

// TransitionUnit validates a transition from current to next status.
func TransitionUnit(current, next UnitStatus) error {
	if !CanTransition(current, next) {
		return ErrInvalidTransition
	}
	return nil
}
