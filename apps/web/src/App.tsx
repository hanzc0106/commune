import * as Tabs from "@radix-ui/react-tabs";
import { useEffect, useState, type FormEvent, type ReactNode } from "react";

import {
  createTransaction,
  getBootstrap,
  getMonthlyOverview,
  initializeApp,
  listCategories,
  listLoginMembers,
  listTransactions,
  login,
  logout,
  type Category,
  type MonthlyOverview,
  type Transaction,
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

const today = () => new Date().toISOString().slice(0, 10);
const currentMonth = () => new Date().toISOString().slice(0, 7);

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
  const [month, setMonth] = useState(currentMonth());
  const [categories, setCategories] = useState<Category[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [overview, setOverview] = useState<MonthlyOverview | null>(null);
  const [ledgerError, setLedgerError] = useState("");
  const [loadingLedger, setLoadingLedger] = useState(true);

  async function refreshLedger(targetMonth = month) {
    setLedgerError("");
    setLoadingLedger(true);
    try {
      const [categoryResult, transactionResult, overviewResult] = await Promise.all([
        listCategories(),
        listTransactions(targetMonth),
        getMonthlyOverview(targetMonth)
      ]);
      setCategories(categoryResult.categories);
      setTransactions(transactionResult.transactions);
      setOverview(overviewResult);
    } catch (error) {
      setLedgerError(error instanceof Error ? error.message : "加载账本失败");
    } finally {
      setLoadingLedger(false);
    }
  }

  useEffect(() => {
    let cancelled = false;
    setLedgerError("");
    setLoadingLedger(true);
    Promise.all([listCategories(), listTransactions(month), getMonthlyOverview(month)])
      .then(([categoryResult, transactionResult, overviewResult]) => {
        if (cancelled) {
          return;
        }
        setCategories(categoryResult.categories);
        setTransactions(transactionResult.transactions);
        setOverview(overviewResult);
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          setLedgerError(error instanceof Error ? error.message : "加载账本失败");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoadingLedger(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [month]);

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
                <AddTransactionPanel
                  categories={categories}
                  onCreated={() => refreshLedger(month)}
                />
                <MonthlyOverviewPanel
                  overview={overview}
                  loading={loadingLedger}
                  error={ledgerError}
                />
              </div>
            </Tabs.Content>

            <Tabs.Content value="transactions">
              <TransactionsPanel
                month={month}
                onMonthChange={setMonth}
                transactions={transactions}
                loading={loadingLedger}
                error={ledgerError}
              />
            </Tabs.Content>

            <Tabs.Content value="budgets">
              <Panel eyebrow="预算" title="预算功能待接入" body="账本流水已经接入，下一阶段会在这里配置分类预算和查看使用进度。" />
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

function AddTransactionPanel({
  categories,
  onCreated
}: {
  categories: Category[];
  onCreated: () => Promise<void>;
}) {
  const [type, setType] = useState<"expense" | "income">("expense");
  const [amount, setAmount] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [transactionDate, setTransactionDate] = useState(today());
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const availableCategories = categories.filter((category) => category.type === type);
  const selectedCategoryId = categoryId || availableCategories[0]?.id || "";

  useEffect(() => {
    if (!availableCategories.some((category) => category.id === categoryId)) {
      setCategoryId(availableCategories[0]?.id ?? "");
    }
  }, [availableCategories, categoryId]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    const amountCents = yuanToCents(amount);
    if (amountCents <= 0) {
      setError("请输入有效金额");
      return;
    }
    if (!selectedCategoryId) {
      setError("请选择分类");
      return;
    }
    setSubmitting(true);
    try {
      await createTransaction({
        type,
        amountCents,
        categoryId: selectedCategoryId,
        transactionDate,
        note
      });
      setAmount("");
      setNote("");
      await onCreated();
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "保存失败");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm md:p-5">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-slate-600">快速记账</p>
          <h2 className="mt-1 text-2xl font-semibold">记一笔</h2>
        </div>
        <div className="grid grid-cols-2 rounded-lg bg-slate-100 p-1 text-sm">
          {(["expense", "income"] as const).map((item) => (
            <button
              key={item}
              type="button"
              onClick={() => setType(item)}
              className={`rounded-md px-3 py-2 ${type === item ? "bg-white text-emerald-700 shadow-sm" : "text-slate-600"}`}
            >
              {item === "expense" ? "支出" : "收入"}
            </button>
          ))}
        </div>
      </div>

      <label className="mt-5 block">
        <span className="text-sm font-medium text-slate-700">金额</span>
        <input
          value={amount}
          onChange={(event) => setAmount(event.target.value)}
          inputMode="decimal"
          placeholder="0.00"
          className="mt-2 w-full rounded-md border border-slate-300 px-3 py-4 text-3xl font-semibold outline-none focus:border-emerald-700"
        />
      </label>

      <div className="mt-5">
        <p className="text-sm font-medium text-slate-700">分类</p>
        <div className="mt-2 grid grid-cols-2 gap-2 sm:grid-cols-3">
          {availableCategories.map((category) => (
            <button
              key={category.id}
              type="button"
              onClick={() => setCategoryId(category.id)}
              className={`rounded-md border px-3 py-3 text-left text-sm ${
                selectedCategoryId === category.id
                  ? "border-emerald-700 bg-emerald-50 text-emerald-800"
                  : "border-slate-200 bg-white text-slate-700"
              }`}
            >
              {category.name}
            </button>
          ))}
        </div>
      </div>

      <div className="mt-5 grid gap-3 sm:grid-cols-2">
        <label className="block">
          <span className="text-sm font-medium text-slate-700">日期</span>
          <input
            type="date"
            value={transactionDate}
            onChange={(event) => setTransactionDate(event.target.value)}
            className="mt-2 w-full rounded-md border border-slate-300 px-3 py-3 outline-none focus:border-emerald-700"
          />
        </label>
        <label className="block">
          <span className="text-sm font-medium text-slate-700">备注</span>
          <input
            value={note}
            onChange={(event) => setNote(event.target.value)}
            className="mt-2 w-full rounded-md border border-slate-300 px-3 py-3 outline-none focus:border-emerald-700"
          />
        </label>
      </div>

      {error ? <p className="mt-4 text-sm text-red-600">{error}</p> : null}
      <button
        type="submit"
        disabled={submitting}
        className="mt-5 w-full rounded-md bg-emerald-700 px-4 py-3 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-60"
      >
        {submitting ? "正在保存" : "保存"}
      </button>
    </form>
  );
}

function MonthlyOverviewPanel({
  overview,
  loading,
  error
}: {
  overview: MonthlyOverview | null;
  loading: boolean;
  error: string;
}) {
  if (loading) {
    return <Panel eyebrow="本月概览" title="加载中" body="正在读取本月流水。" />;
  }
  if (error) {
    return <Panel eyebrow="本月概览" title="加载失败" body={error} />;
  }
  if (!overview) {
    return <Panel eyebrow="本月概览" title="暂无数据" body="保存第一笔流水后会显示本月汇总。" />;
  }
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm md:p-5">
      <p className="text-sm font-medium text-slate-600">本月概览</p>
      <div className="mt-4 grid grid-cols-3 gap-2">
        <Metric label="收入" value={formatMoney(overview.incomeCents)} />
        <Metric label="支出" value={formatMoney(overview.expenseCents)} />
        <Metric label="结余" value={formatMoney(overview.balanceCents)} />
      </div>
      <div className="mt-5 space-y-3">
        {overview.categoryTotals.slice(0, 3).map((item) => (
          <div key={item.categoryId} className="flex items-center justify-between text-sm">
            <span className="text-slate-600">{item.categoryName}</span>
            <span className="font-medium">{formatMoney(item.expenseCents)}</span>
          </div>
        ))}
        {overview.categoryTotals.length === 0 ? <p className="text-sm text-slate-500">暂无支出分类数据</p> : null}
      </div>
    </div>
  );
}

function TransactionsPanel({
  month,
  onMonthChange,
  transactions,
  loading,
  error
}: {
  month: string;
  onMonthChange: (month: string) => void;
  transactions: Transaction[];
  loading: boolean;
  error: string;
}) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm md:p-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="text-sm font-medium text-slate-600">流水</p>
          <h2 className="mt-1 text-2xl font-semibold">家庭流水</h2>
        </div>
        <input
          type="month"
          value={month}
          onChange={(event) => onMonthChange(event.target.value)}
          className="rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-emerald-700"
        />
      </div>
      {loading ? <p className="mt-5 text-sm text-slate-500">正在加载...</p> : null}
      {error ? <p className="mt-5 text-sm text-red-600">{error}</p> : null}
      {!loading && !error && transactions.length === 0 ? (
        <p className="mt-5 text-sm text-slate-500">这个月份还没有流水。</p>
      ) : null}
      <div className="mt-4 divide-y divide-slate-100">
        {transactions.map((transaction) => (
          <div key={transaction.id} className="flex items-center justify-between gap-4 py-3">
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{transaction.category.name}</p>
              <p className="mt-1 text-xs text-slate-500">
                {transaction.transactionDate} · {transaction.member.name}
                {transaction.note ? ` · ${transaction.note}` : ""}
              </p>
            </div>
            <p className={`text-sm font-semibold ${transaction.type === "income" ? "text-emerald-700" : "text-slate-950"}`}>
              {transaction.type === "income" ? "+" : "-"}
              {formatMoney(transaction.amountCents)}
            </p>
          </div>
        ))}
      </div>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md bg-slate-50 p-3">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 text-sm font-semibold">{value}</p>
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

function yuanToCents(value: string): number {
  const normalized = value.trim();
  if (!/^\d+(\.\d{1,2})?$/.test(normalized)) {
    return 0;
  }
  const [yuan, cents = ""] = normalized.split(".");
  return Number(yuan) * 100 + Number(cents.padEnd(2, "0"));
}

function formatMoney(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}
