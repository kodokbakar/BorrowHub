import { api } from "./client";
import type {
  ApprovedRequestResult,
  AuthResult,
  Category,
  ConditionLog,
  Item,
  ItemAvailability,
  Loan,
  LoanRequest,
  LoanStatus,
  MembershipDetail,
  MeResponse,
  Notification,
  NotificationStatus,
  Organization,
  RequestStatus,
  ReturnCondition,
  Tokens,
  Unit,
  UnitCondition,
  UnitStatus,
  UserOrganization,
} from "./types";

export const identityApi = {
  login(email: string, password: string): Promise<AuthResult> {
    return api.post<AuthResult>("identity", "/api/v1/auth/login", {
      email,
      password,
    });
  },

  register(
    email: string,
    password: string,
    displayName: string,
    termsAccepted: boolean,
  ): Promise<AuthResult> {
    return api.post<AuthResult>("identity", "/api/v1/auth/register", {
      email,
      password,
      display_name: displayName,
      terms_accepted: termsAccepted,
    });
  },

  refresh(refreshToken: string): Promise<Tokens> {
    return api.post<Tokens>("identity", "/api/v1/auth/refresh", {
      refresh_token: refreshToken,
    });
  },

  logout(refreshToken?: string): Promise<{ message: string }> {
    return api.post<{ message: string }>("identity", "/api/v1/auth/logout", {
      refresh_token: refreshToken,
    });
  },

  getMe(): Promise<MeResponse> {
    return api.get<MeResponse>("identity", "/api/v1/me");
  },

  createOrganization(data: {
    name: string;
    slug: string;
    timezone?: string;
    visibility?: string;
    max_loan_duration_days?: number;
  }): Promise<Organization> {
    return api.post<Organization>("identity", "/api/v1/organizations", {
      timezone: "UTC",
      visibility: "public",
      max_loan_duration_days: 14,
      ...data,
    });
  },

  listMembers(organizationId: string): Promise<MembershipDetail[]> {
    return api.get<MembershipDetail[]>(
      "identity",
      `/api/v1/organizations/${organizationId}/members`,
    );
  },
};

export const inventoryApi = {
  listCategories(organizationId: string): Promise<Category[]> {
    return api.get<Category[]>(
      "inventory",
      `/api/v1/organizations/${organizationId}/categories`,
      undefined,
      { organizationId },
    );
  },

  createCategory(
    organizationId: string,
    data: { name: string; slug: string; description?: string },
  ): Promise<Category> {
    return api.post<Category>(
      "inventory",
      `/api/v1/organizations/${organizationId}/categories`,
      data,
      { organizationId },
    );
  },

  updateCategory(
    categoryId: string,
    data: { name?: string; slug?: string; description?: string },
  ): Promise<Category> {
    return api.patch<Category>(
      "inventory",
      `/api/v1/categories/${categoryId}`,
      data,
    );
  },

  deleteCategory(categoryId: string): Promise<{ message: string }> {
    return api.delete<{ message: string }>(
      "inventory",
      `/api/v1/categories/${categoryId}`,
    );
  },

  listItems(organizationId: string): Promise<Item[]> {
    return api.get<Item[]>(
      "inventory",
      `/api/v1/organizations/${organizationId}/items`,
      undefined,
      { organizationId },
    );
  },

  getItem(itemId: string): Promise<Item> {
    return api.get<Item>("inventory", `/api/v1/items/${itemId}`);
  },

  createItem(
    organizationId: string,
    data: {
      category_id: string;
      name: string;
      description?: string;
      is_lendable?: boolean;
    },
  ): Promise<Item> {
    return api.post<Item>(
      "inventory",
      `/api/v1/organizations/${organizationId}/items`,
      data,
      { organizationId },
    );
  },

  updateItem(
    itemId: string,
    data: {
      category_id?: string;
      name?: string;
      description?: string;
      is_lendable?: boolean;
    },
  ): Promise<Item> {
    return api.patch<Item>("inventory", `/api/v1/items/${itemId}`, data);
  },

  deleteItem(itemId: string): Promise<{ message: string }> {
    return api.delete<{ message: string }>("inventory", `/api/v1/items/${itemId}`);
  },

  listUnits(itemId: string): Promise<Unit[]> {
    return api.get<Unit[]>("inventory", `/api/v1/items/${itemId}/units`);
  },

  getUnit(unitId: string): Promise<Unit> {
    return api.get<Unit>("inventory", `/api/v1/units/${unitId}`);
  },

  createUnit(
    itemId: string,
    data: {
      barcode: string;
      serial_number?: string;
      status?: UnitStatus;
      condition?: UnitCondition;
      notes?: string;
    },
  ): Promise<Unit> {
    return api.post<Unit>("inventory", `/api/v1/items/${itemId}/units`, data);
  },

  updateUnit(
    unitId: string,
    data: {
      barcode?: string;
      serial_number?: string;
      status?: UnitStatus;
      condition?: UnitCondition;
      notes?: string;
      reason?: string;
    },
  ): Promise<Unit> {
    return api.patch<Unit>("inventory", `/api/v1/units/${unitId}`, data);
  },

  getConditionLogs(unitId: string): Promise<ConditionLog[]> {
    return api.get<ConditionLog[]>(
      "inventory",
      `/api/v1/units/${unitId}/condition-logs`,
    );
  },
};

export const lendingApi = {
  getAvailability(
    itemId: string,
    organizationId: string,
    startTime: string,
    endTime: string,
  ): Promise<ItemAvailability> {
    return api.get<ItemAvailability>(
      "lending",
      `/api/v1/items/${itemId}/availability`,
      {
        organization_id: organizationId,
        start_time: startTime,
        end_time: endTime,
      },
      { organizationId },
    );
  },

  createLoanRequest(data: {
    organization_id: string;
    item_id: string;
    quantity: number;
    start_time: string;
    end_time: string;
    purpose: string;
    notes?: string;
  }): Promise<LoanRequest> {
    return api.post<LoanRequest>("lending", "/api/v1/loan-requests", data, {
      organizationId: data.organization_id,
    });
  },

  listLoanRequests(
    organizationId: string,
    status?: RequestStatus,
  ): Promise<LoanRequest[]> {
    return api.get<LoanRequest[]>(
      "lending",
      "/api/v1/loan-requests",
      { organization_id: organizationId, status },
      { organizationId },
    );
  },

  getLoanRequest(requestId: string): Promise<LoanRequest> {
    return api.get<LoanRequest>("lending", `/api/v1/loan-requests/${requestId}`);
  },

  approveLoanRequest(
    requestId: string,
    reviewNotes?: string,
  ): Promise<ApprovedRequestResult> {
    return api.post<ApprovedRequestResult>(
      "lending",
      `/api/v1/loan-requests/${requestId}/approve`,
      { review_notes: reviewNotes },
    );
  },

  rejectLoanRequest(
    requestId: string,
    reviewNotes?: string,
  ): Promise<LoanRequest> {
    return api.post<LoanRequest>(
      "lending",
      `/api/v1/loan-requests/${requestId}/reject`,
      { review_notes: reviewNotes },
    );
  },

  cancelLoanRequest(requestId: string, reason?: string): Promise<LoanRequest> {
    return api.post<LoanRequest>(
      "lending",
      `/api/v1/loan-requests/${requestId}/cancel`,
      { reason },
    );
  },

  listLoans(organizationId: string, status?: LoanStatus): Promise<Loan[]> {
    return api.get<Loan[]>(
      "lending",
      "/api/v1/loans",
      { organization_id: organizationId, status },
      { organizationId },
    );
  },

  getLoan(loanId: string): Promise<Loan> {
    return api.get<Loan>("lending", `/api/v1/loans/${loanId}`);
  },

  checkoutLoan(loanId: string, token: string): Promise<Loan> {
    return api.post<Loan>("lending", `/api/v1/loans/${loanId}/check-out`, {
      token,
    });
  },

  returnLoan(
    loanId: string,
    returnCondition: ReturnCondition,
    returnNotes?: string,
  ): Promise<Loan> {
    return api.post<Loan>("lending", `/api/v1/loans/${loanId}/return`, {
      return_condition: returnCondition,
      return_notes: returnNotes,
    });
  },

  listOverdue(organizationId: string): Promise<Loan[]> {
    return api.get<Loan[]>(
      "lending",
      `/api/v1/organizations/${organizationId}/overdue`,
      undefined,
      { organizationId },
    );
  },
};

export const notificationApi = {
  listNotifications(
    organizationId: string,
    status?: NotificationStatus,
    limit = 50,
    offset = 0,
  ): Promise<Notification[]> {
    return api.get<Notification[]>(
      "notification",
      `/api/v1/organizations/${organizationId}/notifications`,
      {
        status,
        limit,
        offset,
      },
      { organizationId },
    );
  },

  markAsRead(
    notificationId: string,
    organizationId: string,
  ): Promise<Notification> {
    return api.patch<Notification>(
      "notification",
      `/api/v1/notifications/${notificationId}/read`,
      undefined,
      {
        params: { organization_id: organizationId },
        organizationId,
      },
    );
  },
};
