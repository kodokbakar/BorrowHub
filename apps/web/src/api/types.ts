export type Role = "owner" | "admin" | "member";

export interface User {
  id: string;
  email: string;
  display_name: string;
  status: string;
  terms_accepted_at: string;
  created_at: string;
  updated_at: string;
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  timezone: string;
  visibility: string;
  max_loan_duration_days: number;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface UserOrganization extends Organization {
  role: Role;
}

export interface Membership {
  id: string;
  organization_id: string;
  user_id: string;
  role: Role;
  created_at: string;
  updated_at: string;
}

export interface MembershipDetail extends Membership {
  email: string;
  display_name: string;
}

export interface Tokens {
  access_token: string;
  refresh_token: string;
  token_type?: string;
  expires_in?: number;
}

export interface AuthResult {
  user: User;
  tokens: Tokens;
}

export interface MeResponse {
  user: User;
  organizations: UserOrganization[];
}

export type UnitStatus =
  | "available"
  | "reserved"
  | "borrowed"
  | "maintenance"
  | "retired";

export type UnitCondition = "new" | "good" | "fair" | "poor" | "damaged";

export interface Category {
  id: string;
  organization_id: string;
  name: string;
  slug: string;
  description?: string | null;
  created_at: string;
  updated_at: string;
}

export interface Item {
  id: string;
  organization_id: string;
  category_id: string;
  name: string;
  description?: string | null;
  is_lendable: boolean;
  created_at: string;
  updated_at: string;
}

export interface Unit {
  id: string;
  organization_id: string;
  item_id: string;
  barcode: string;
  serial_number?: string | null;
  status: UnitStatus;
  condition: UnitCondition;
  notes?: string | null;
  created_at: string;
  updated_at: string;
}

export interface ConditionLog {
  id: string;
  unit_id: string;
  previous_condition?: UnitCondition | null;
  new_condition: UnitCondition;
  reason?: string | null;
  changed_by: string;
  created_at: string;
}

export type RequestStatus = "pending" | "approved" | "rejected" | "cancelled";

export type LoanStatus =
  | "pending_checkout"
  | "checked_out"
  | "returned"
  | "overdue"
  | "cancelled";

export type ReturnCondition = "new" | "good" | "fair" | "poor" | "damaged";

export interface LoanRequest {
  id: string;
  organization_id: string;
  requester_user_id: string;
  item_id: string;
  quantity: number;
  start_time: string;
  end_time: string;
  purpose: string;
  notes?: string | null;
  status: RequestStatus;
  reviewer_user_id?: string | null;
  review_notes?: string | null;
  reviewed_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface Loan {
  id: string;
  organization_id: string;
  loan_request_id: string;
  reservation_id: string;
  unit_id: string;
  borrower_user_id: string;
  status: LoanStatus;
  checkout_time?: string | null;
  expected_return_time: string;
  actual_return_time?: string | null;
  return_condition?: ReturnCondition | null;
  return_notes?: string | null;
  processed_by_user_id?: string | null;
  created_at: string;
  updated_at: string;
}

export interface ItemAvailability {
  item_id: string;
  start_time: string;
  end_time: string;
  total_units: number;
  available_units: number;
  is_available: boolean;
  eligible_unit_ids?: string[];
}

export interface ApprovedRequestResult {
  loan_request: LoanRequest;
  loan: Loan;
  checkout_token: string;
}

export type NotificationStatus = "unread" | "read";

export interface Notification {
  id: string;
  organization_id: string;
  recipient_user_id: string;
  event_id: string;
  event_type: string;
  title: string;
  message: string;
  data: Record<string, unknown>;
  status: NotificationStatus;
  read_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface Delivery {
  id: string;
  notification_id: string;
  channel: "in_app" | "email";
  recipient: string;
  status: "pending" | "delivered" | "failed" | "retrying";
  attempts_count: number;
  last_error?: string | null;
  created_at: string;
  updated_at: string;
}

export interface ApiMeta {
  request_id?: string;
  total?: number;
  limit?: number;
  offset?: number;
}

export interface ApiSuccessEnvelope<T> {
  data: T;
  meta?: ApiMeta;
}

export interface ApiErrorDetail {
  code: string;
  message: string;
  fields?: Record<string, string>;
  request_id?: string;
}

export interface ApiErrorEnvelope {
  error: ApiErrorDetail;
}
