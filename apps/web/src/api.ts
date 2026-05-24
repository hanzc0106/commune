export type Member = {
  id: string;
  name: string;
  role: "admin" | "member";
};

export type MemberAdmin = Member & {
  active: boolean;
};

export type BootstrapResponse = {
  initialized: boolean;
  householdName: string;
  session: null | {
    member: Member;
  };
};

export type Category = {
  id: string;
  name: string;
  type: "expense" | "income";
  iconKey: string;
  colorKey: string;
  sortOrder: number;
  systemDefault: boolean;
};

export type Transaction = {
  id: string;
  type: "expense" | "income";
  amountCents: number;
  category: Category;
  member: Member;
  transactionDate: string;
  note: string;
  createdAt: string;
  updatedAt: string;
};

export type MonthlyOverview = {
  month: string;
  incomeCents: number;
  expenseCents: number;
  balanceCents: number;
  categoryTotals: Array<{
    categoryId: string;
    categoryName: string;
    iconKey: string;
    colorKey: string;
    expenseCents: number;
  }>;
  recent: Transaction[];
};

export type TransactionInput = {
  type: "expense" | "income";
  amountCents: number;
  categoryId: string;
  transactionDate: string;
  note: string;
};

export type CreateMemberInput = {
  name: string;
  role: "admin" | "member";
  pin: string;
};

export type CreateCategoryInput = {
  name: string;
  type: "expense" | "income";
  iconKey: string;
  colorKey: string;
};

export type UpdateCategoryInput = {
  name: string;
  iconKey: string;
  colorKey: string;
  sortOrder: number;
};

export async function getBootstrap(): Promise<BootstrapResponse> {
  return request<BootstrapResponse>("/api/bootstrap");
}

export async function initializeApp(input: {
  householdName: string;
  adminName: string;
  pin: string;
}): Promise<{ member: Member }> {
  return request<{ member: Member }>("/api/init", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function listLoginMembers(): Promise<{ members: Array<{ id: string; name: string }> }> {
  return request<{ members: Array<{ id: string; name: string }> }>("/api/login-members");
}

export async function login(input: { memberId: string; pin: string }): Promise<{ member: Member }> {
  return request<{ member: Member }>("/api/login", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function logout(): Promise<{ ok: true }> {
  return request<{ ok: true }>("/api/logout", {
    method: "POST",
    body: JSON.stringify({})
  });
}

export async function listCategories(): Promise<{ categories: Category[] }> {
  return request<{ categories: Category[] }>("/api/categories");
}

export async function listMembers(): Promise<{ members: MemberAdmin[] }> {
  return request<{ members: MemberAdmin[] }>("/api/members");
}

export async function createMember(input: CreateMemberInput): Promise<MemberAdmin> {
  return request<MemberAdmin>("/api/members", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function disableMember(id: string): Promise<MemberAdmin> {
  return request<MemberAdmin>(`/api/members/${encodeURIComponent(id)}/disable`, {
    method: "POST",
    body: JSON.stringify({})
  });
}

export async function resetMemberPIN(id: string, pin: string): Promise<{ ok: true }> {
  return request<{ ok: true }>(`/api/members/${encodeURIComponent(id)}/reset-pin`, {
    method: "POST",
    body: JSON.stringify({ pin })
  });
}

export async function changeOwnPIN(input: { currentPin: string; newPin: string }): Promise<{ ok: true }> {
  return request<{ ok: true }>("/api/me/change-pin", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function createCategory(input: CreateCategoryInput): Promise<Category> {
  return request<Category>("/api/categories", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function updateCategory(id: string, input: UpdateCategoryInput): Promise<Category> {
  return request<Category>(`/api/categories/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify(input)
  });
}

export async function disableCategory(id: string): Promise<Category> {
  return request<Category>(`/api/categories/${encodeURIComponent(id)}/disable`, {
    method: "POST",
    body: JSON.stringify({})
  });
}

export async function listTransactions(month: string): Promise<{ transactions: Transaction[] }> {
  return request<{ transactions: Transaction[] }>(`/api/transactions?month=${encodeURIComponent(month)}`);
}

export async function createTransaction(input: TransactionInput): Promise<Transaction> {
  return request<Transaction>("/api/transactions", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function updateTransaction(id: string, input: TransactionInput): Promise<Transaction> {
  return request<Transaction>(`/api/transactions/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify(input)
  });
}

export async function deleteTransaction(id: string): Promise<{ ok: true }> {
  return request<{ ok: true }>(`/api/transactions/${encodeURIComponent(id)}`, {
    method: "DELETE"
  });
}

export async function getMonthlyOverview(month: string): Promise<MonthlyOverview> {
  return request<MonthlyOverview>(`/api/overview/monthly?month=${encodeURIComponent(month)}`);
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...init?.headers
    }
  });
  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.error ?? "请求失败");
  }
  return data as T;
}
