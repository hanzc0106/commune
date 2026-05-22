import type { BootstrapResponse, Member } from "./api";

export type AppState =
  | { status: "loading" }
  | { status: "needs-init" }
  | { status: "needs-login"; householdName: string }
  | { status: "authenticated"; householdName: string; member: Member }
  | { status: "error"; message: string };

export function bootstrapToState(bootstrap: BootstrapResponse): AppState {
  if (!bootstrap.initialized) {
    return { status: "needs-init" };
  }
  if (bootstrap.session?.member) {
    return {
      status: "authenticated",
      householdName: bootstrap.householdName,
      member: bootstrap.session.member
    };
  }
  return {
    status: "needs-login",
    householdName: bootstrap.householdName
  };
}
