package domain

import (
	"fmt"
)

func IsAdminOrOwner(role string) bool {
	return role == "admin" || role == "owner"
}

func ValidateRequestTransition(from, to RequestStatus, role string, isRequester bool) error {
	if from == to {
		return fmt.Errorf("%w: cannot transition from %s to itself", ErrInvalidTransition, from)
	}

	switch from {
	case RequestStatusPending:
		switch to {
		case RequestStatusApproved:
			if !IsAdminOrOwner(role) {
				return fmt.Errorf("%w: only admins can approve loan requests", ErrUnauthorizedAction)
			}
			return nil
		case RequestStatusRejected:
			if !IsAdminOrOwner(role) {
				return fmt.Errorf("%w: only admins can reject loan requests", ErrUnauthorizedAction)
			}
			return nil
		case RequestStatusCancelled:
			if !isRequester && !IsAdminOrOwner(role) {
				return fmt.Errorf("%w: only requester or admin can cancel pending loan request", ErrUnauthorizedAction)
			}
			return nil
		default:
			return fmt.Errorf("%w: invalid transition from %s to %s", ErrInvalidTransition, from, to)
		}
	case RequestStatusApproved:
		if to == RequestStatusCancelled {
			if !IsAdminOrOwner(role) {
				return fmt.Errorf("%w: only admins can cancel an approved loan request", ErrUnauthorizedAction)
			}
			return nil
		}
		return fmt.Errorf("%w: cannot transition from approved to %s", ErrInvalidTransition, to)
	case RequestStatusRejected:
		return fmt.Errorf("%w: rejected request is terminal", ErrInvalidTransition)
	case RequestStatusCancelled:
		return fmt.Errorf("%w: cancelled request is terminal", ErrInvalidTransition)
	default:
		return fmt.Errorf("%w: unknown initial status %s", ErrInvalidTransition, from)
	}
}

func ValidateLoanTransition(from, to LoanStatus) error {
	if from == to {
		return fmt.Errorf("%w: cannot transition from %s to itself", ErrInvalidTransition, from)
	}

	switch from {
	case LoanStatusPendingCheckout:
		if to == LoanStatusCheckedOut || to == LoanStatusCancelled {
			return nil
		}
		return fmt.Errorf("%w: cannot transition loan from %s to %s", ErrInvalidTransition, from, to)
	case LoanStatusCheckedOut:
		if to == LoanStatusReturned || to == LoanStatusOverdue {
			return nil
		}
		return fmt.Errorf("%w: cannot transition loan from %s to %s", ErrInvalidTransition, from, to)
	case LoanStatusOverdue:
		if to == LoanStatusReturned {
			return nil
		}
		return fmt.Errorf("%w: cannot transition loan from %s to %s", ErrInvalidTransition, from, to)
	case LoanStatusReturned:
		return fmt.Errorf("%w: returned loan is terminal", ErrInvalidTransition)
	case LoanStatusCancelled:
		return fmt.Errorf("%w: cancelled loan is terminal", ErrInvalidTransition)
	default:
		return fmt.Errorf("%w: unknown initial loan status %s", ErrInvalidTransition, from)
	}
}

func ValidateReservationTransition(from, to ReservationStatus) error {
	if from == to {
		return fmt.Errorf("%w: cannot transition from %s to itself", ErrInvalidTransition, from)
	}

	if from == ReservationStatusActive {
		if to == ReservationStatusFulfilled || to == ReservationStatusCancelled {
			return nil
		}
	}
	return fmt.Errorf("%w: cannot transition reservation from %s to %s", ErrInvalidTransition, from, to)
}
