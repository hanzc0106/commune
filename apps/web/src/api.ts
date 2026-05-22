export type Member = {
  id: string;
  name: string;
  role: "admin" | "member";
};

export type BootstrapResponse = {
  initialized: boolean;
  householdName: string;
  session: null | {
    member: Member;
  };
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
