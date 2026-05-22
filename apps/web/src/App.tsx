import * as Tabs from "@radix-ui/react-tabs";
import { useEffect, useState, type FormEvent, type ReactNode } from "react";

import {
  getBootstrap,
  initializeApp,
  listLoginMembers,
  login,
  logout,
  type Member
} from "./api";
import { bootstrapToState, type AppState } from "./auth";

type AppProps = {
  icon: ReactNode;
};

const navItems = [
  { value: "add", label: "记一笔" },
  { value: "transactions", label: "流水" },
  { value: "budgets", label: "预算" },
  { value: "settings", label: "设置" }
];

export function App({ icon }: AppProps) {
  const [state, setState] = useState<AppState>({ status: "loading" });

  useEffect(() => {
    let cancelled = false;
    getBootstrap()
      .then((bootstrap) => {
        if (!cancelled) {
          setState(bootstrapToState(bootstrap));
        }
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          setState({
            status: "error",
            message: error instanceof Error ? error.message : "加载失败"
          });
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  if (state.status === "loading") {
    return <LoadingScreen />;
  }
  if (state.status === "error") {
    return <CenteredMessage title="无法加载 Commune" body={state.message} />;
  }
  if (state.status === "needs-init") {
    return (
      <InitScreen
        icon={icon}
        onInitialized={(member, householdName) => {
          setState({ status: "authenticated", householdName, member });
        }}
      />
    );
  }
  if (state.status === "needs-login") {
    return (
      <LoginScreen
        householdName={state.householdName}
        onLogin={(member) => {
          setState({ status: "authenticated", householdName: state.householdName, member });
        }}
      />
    );
  }
  return (
    <AuthenticatedShell
      icon={icon}
      member={state.member}
      householdName={state.householdName}
      onLogout={async () => {
        await logout();
        setState({ status: "needs-login", householdName: state.householdName });
      }}
    />
  );
}

function LoadingScreen() {
  return <CenteredMessage title="Commune" body="正在加载家庭账本..." />;
}

function CenteredMessage({ title, body }: { title: string; body: string }) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-stone-50 px-4 text-slate-950">
      <section className="w-full max-w-sm rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
        <h1 className="text-xl font-semibold">{title}</h1>
        <p className="mt-2 text-sm leading-6 text-slate-600">{body}</p>
      </section>
    </main>
  );
}

function InitScreen({
  icon,
  onInitialized
}: {
  icon: ReactNode;
  onInitialized: (member: Member, householdName: string) => void;
}) {
  const [householdName, setHouseholdName] = useState("");
  const [adminName, setAdminName] = useState("");
  const [pin, setPin] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      const result = await initializeApp({ householdName, adminName, pin });
      onInitialized(result.member, householdName.trim());
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "初始化失败");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-stone-50 px-4 text-slate-950">
      <form onSubmit={handleSubmit} className="w-full max-w-md rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
        <BrandHeader icon={icon} compact />
        <div className="mt-6">
          <h1 className="text-2xl font-semibold">初始化家庭账本</h1>
          <p className="mt-2 text-sm leading-6 text-slate-600">创建家庭名称和第一个管理员账号。</p>
        </div>
        <FormField label="家庭名称" value={householdName} onChange={setHouseholdName} autoComplete="organization" />
        <FormField label="管理员姓名" value={adminName} onChange={setAdminName} autoComplete="name" />
        <FormField label="管理员 PIN" value={pin} onChange={setPin} type="password" autoComplete="new-password" />
        {error ? <p className="mt-4 text-sm text-red-600">{error}</p> : null}
        <button
          type="submit"
          disabled={submitting}
          className="mt-5 w-full rounded-md bg-emerald-700 px-4 py-3 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-60"
        >
          {submitting ? "正在初始化" : "创建账本"}
        </button>
      </form>
    </main>
  );
}

function LoginScreen({ householdName, onLogin }: { householdName: string; onLogin: (member: Member) => void }) {
  const [members, setMembers] = useState<Array<{ id: string; name: string }>>([]);
  const [memberId, setMemberId] = useState("");
  const [pin, setPin] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    let cancelled = false;
    listLoginMembers()
      .then((result) => {
        if (cancelled) {
          return;
        }
        setMembers(result.members);
        setMemberId(result.members[0]?.id ?? "");
      })
      .catch((loadError: unknown) => {
        if (!cancelled) {
          setError(loadError instanceof Error ? loadError.message : "加载成员失败");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      const result = await login({ memberId, pin });
      onLogin(result.member);
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "登录失败");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-stone-50 px-4 text-slate-950">
      <form onSubmit={handleSubmit} className="w-full max-w-md rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
        <p className="text-sm font-medium text-emerald-700">{householdName}</p>
        <h1 className="mt-2 text-2xl font-semibold">登录家庭账本</h1>
        <label className="mt-5 block">
          <span className="text-sm font-medium text-slate-700">成员</span>
          <select
            value={memberId}
            onChange={(event) => setMemberId(event.target.value)}
            disabled={loading}
            className="mt-2 w-full rounded-md border border-slate-300 bg-white px-3 py-3 text-base outline-none focus:border-emerald-700"
          >
            {members.map((member) => (
              <option key={member.id} value={member.id}>
                {member.name}
              </option>
            ))}
          </select>
        </label>
        <FormField label="PIN" value={pin} onChange={setPin} type="password" autoComplete="current-password" />
        {error ? <p className="mt-4 text-sm text-red-600">{error}</p> : null}
        <button
          type="submit"
          disabled={submitting || loading || !memberId}
          className="mt-5 w-full rounded-md bg-emerald-700 px-4 py-3 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-60"
        >
          {submitting ? "正在登录" : "登录"}
        </button>
      </form>
    </main>
  );
}

function FormField({
  label,
  value,
  onChange,
  type = "text",
  autoComplete
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  autoComplete?: string;
}) {
  return (
    <label className="mt-4 block">
      <span className="text-sm font-medium text-slate-700">{label}</span>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        type={type}
        autoComplete={autoComplete}
        className="mt-2 w-full rounded-md border border-slate-300 px-3 py-3 text-base outline-none focus:border-emerald-700"
      />
    </label>
  );
}

function AuthenticatedShell({
  icon,
  member,
  householdName,
  onLogout
}: {
  icon: ReactNode;
  member: Member;
  householdName: string;
  onLogout: () => void;
}) {
  return (
    <Tabs.Root defaultValue="add" asChild>
      <main className="min-h-screen bg-stone-50 text-slate-950 md:grid md:grid-cols-[15rem_minmax(0,1fr)]">
        <aside className="hidden border-r border-slate-200 bg-white md:flex md:min-h-screen md:flex-col">
          <BrandHeader icon={icon} />
          <Tabs.List className="flex flex-col gap-1 px-3 py-4 text-sm">
            {navItems.map((item) => (
              <Tabs.Trigger
                key={item.value}
                value={item.value}
                className="rounded-md px-3 py-2 text-left text-slate-700 transition-colors data-[state=active]:bg-emerald-700 data-[state=active]:text-white"
              >
                {item.label}
              </Tabs.Trigger>
            ))}
          </Tabs.List>
          <div className="mt-auto border-t border-slate-200 p-4">
            <p className="text-sm font-medium">{member.name}</p>
            <p className="text-xs text-slate-500">{member.role === "admin" ? "管理员" : "成员"}</p>
            <button onClick={onLogout} className="mt-3 text-sm font-medium text-slate-600 hover:text-slate-950">
              退出登录
            </button>
          </div>
        </aside>

        <section className="mx-auto flex min-h-screen w-full max-w-6xl flex-col px-4 pb-24 pt-5 md:px-8 md:pb-8 lg:px-10">
          <header className="md:hidden">
            <BrandHeader icon={icon} compact />
          </header>

          <div className="hidden items-start justify-between gap-6 md:flex">
            <div>
              <p className="text-sm font-medium text-emerald-700">{householdName}</p>
              <h1 className="mt-1 text-2xl font-semibold">家庭共享账本</h1>
            </div>
            <button
              onClick={onLogout}
              className="rounded-lg border border-slate-200 bg-white px-4 py-3 text-sm text-slate-600 shadow-sm hover:text-slate-950"
            >
              {member.name} · 退出
            </button>
          </div>

          <div className="mt-6 flex-1 md:mt-8">
            <Tabs.Content value="add" className="h-full">
              <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_20rem]">
                <Panel
                  eyebrow="下一步"
                  title="建立记账基础流程"
                  body="认证基础已经就绪。后续这里会成为快速记账表单。"
                />
                <Panel
                  eyebrow="当前成员"
                  title={member.name}
                  body={member.role === "admin" ? "你现在以管理员身份登录。" : "你现在以家庭成员身份登录。"}
                />
              </div>
            </Tabs.Content>

            <Tabs.Content value="transactions">
              <Panel eyebrow="流水" title="按月份查看家庭流水" body="后续会在这里放筛选条件、分组列表和编辑入口。" />
            </Tabs.Content>

            <Tabs.Content value="budgets">
              <Panel eyebrow="预算" title="查看分类预算使用情况" body="后续会展示分类预算、已花金额、剩余金额和超支状态。" />
            </Tabs.Content>

            <Tabs.Content value="settings">
              <Panel eyebrow="设置" title="维护家庭成员和分类" body="管理员设置会在 PC 端使用更宽的表单布局，移动端保持单列。" />
            </Tabs.Content>
          </div>

          <Tabs.List className="fixed inset-x-3 bottom-3 grid grid-cols-4 rounded-lg border border-slate-200 bg-white p-1 text-sm shadow-lg md:hidden">
            {navItems.map((item) => (
              <Tabs.Trigger
                key={item.value}
                value={item.value}
                className="rounded-md px-2 py-2 data-[state=active]:bg-emerald-700 data-[state=active]:text-white"
              >
                {item.label}
              </Tabs.Trigger>
            ))}
          </Tabs.List>
        </section>
      </main>
    </Tabs.Root>
  );
}

function BrandHeader({ icon, compact = false }: { icon: ReactNode; compact?: boolean }) {
  return (
    <div className={compact ? "flex items-center gap-3" : "flex items-center gap-3 px-4 py-5"}>
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-emerald-700 text-white">
        {icon}
      </div>
      <div>
        <p className="text-xl font-semibold">Commune</p>
        <p className="text-sm text-slate-600">家庭共享账本</p>
      </div>
    </div>
  );
}

function Panel({ eyebrow, title, body }: { eyebrow: string; title: string; body: string }) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm md:p-5">
      <p className="text-sm font-medium text-slate-600">{eyebrow}</p>
      <p className="mt-2 text-2xl font-semibold md:text-3xl">{title}</p>
      <p className="mt-2 text-sm leading-6 text-slate-600">{body}</p>
    </div>
  );
}
