import * as Tabs from "@radix-ui/react-tabs";
import { useEffect, useState, type FormEvent, type ReactNode } from "react";

import {
  changeOwnPIN,
  copyPreviousBudgets,
  createCategory,
  createMember,
  createTransaction,
  disableCategory,
  disableMember,
  getBudgets,
  getBootstrap,
  getMonthlyOverview,
  initializeApp,
  listCategories,
  listLoginMembers,
  listMembers,
  listTransactions,
  login,
  logout,
  resetMemberPIN,
  setBudget,
  updateCategory,
  type BudgetItem,
  type BudgetSummary,
  type Category,
  type CreateCategoryInput,
  type CreateMemberInput,
  type MemberAdmin,
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
  const [budgetSummary, setBudgetSummary] = useState<BudgetSummary | null>(null);
  const [ledgerError, setLedgerError] = useState("");
  const [budgetError, setBudgetError] = useState("");
  const [loadingLedger, setLoadingLedger] = useState(true);
  const [loadingBudgets, setLoadingBudgets] = useState(true);

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

  async function refreshBudgets(targetMonth = month) {
    setBudgetError("");
    setLoadingBudgets(true);
    try {
      const result = await getBudgets(targetMonth);
      setBudgetSummary(result);
    } catch (error) {
      setBudgetError(error instanceof Error ? error.message : "加载预算失败");
    } finally {
      setLoadingBudgets(false);
    }
  }

  useEffect(() => {
    let cancelled = false;
    setLedgerError("");
    setBudgetError("");
    setLoadingLedger(true);
    setLoadingBudgets(true);
    Promise.all([listCategories(), listTransactions(month), getMonthlyOverview(month), getBudgets(month)])
      .then(([categoryResult, transactionResult, overviewResult, budgetResult]) => {
        if (cancelled) {
          return;
        }
        setCategories(categoryResult.categories);
        setTransactions(transactionResult.transactions);
        setOverview(overviewResult);
        setBudgetSummary(budgetResult);
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          setLedgerError(error instanceof Error ? error.message : "加载账本失败");
          setBudgetError(error instanceof Error ? error.message : "加载预算失败");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoadingLedger(false);
          setLoadingBudgets(false);
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
                  onCreated={async () => {
                    await refreshLedger(month);
                    await refreshBudgets(month);
                  }}
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
              <BudgetsPanel
                member={member}
                month={month}
                onMonthChange={setMonth}
                summary={budgetSummary}
                loading={loadingBudgets}
                error={budgetError}
                onChanged={() => refreshBudgets(month)}
              />
            </Tabs.Content>

            <Tabs.Content value="settings">
              <SettingsPanel
                member={member}
                categories={categories}
                onCategoriesChanged={() => refreshLedger(month)}
              />
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

function BudgetsPanel({
  member,
  month,
  onMonthChange,
  summary,
  loading,
  error,
  onChanged
}: {
  member: Member;
  month: string;
  onMonthChange: (month: string) => void;
  summary: BudgetSummary | null;
  loading: boolean;
  error: string;
  onChanged: () => Promise<void>;
}) {
  const [amounts, setAmounts] = useState<Record<string, string>>({});
  const [message, setMessage] = useState("");
  const [submitError, setSubmitError] = useState("");

  useEffect(() => {
    if (!summary) {
      return;
    }
    setAmounts((current) => {
      const next = { ...current };
      for (const item of summary.items) {
        next[item.category.id] = item.budgetCents > 0 ? centsToYuanInput(item.budgetCents) : "";
      }
      return next;
    });
  }, [summary]);

  async function handleSave(item: BudgetItem) {
    setMessage("");
    setSubmitError("");
    const amountCents = yuanToCents(amounts[item.category.id] ?? "");
    if (amountCents <= 0) {
      setSubmitError("请输入有效预算金额");
      return;
    }
    try {
      await setBudget(month, item.category.id, amountCents);
      await onChanged();
      setMessage("预算已保存");
    } catch (saveError) {
      setSubmitError(saveError instanceof Error ? saveError.message : "保存预算失败");
    }
  }

  async function handleCopyPrevious() {
    setMessage("");
    setSubmitError("");
    try {
      const result = await copyPreviousBudgets(month);
      await onChanged();
      setMessage(`已复制 ${result.copiedCount} 个预算`);
    } catch (copyError) {
      setSubmitError(copyError instanceof Error ? copyError.message : "复制预算失败");
    }
  }

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_22rem]">
      <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm md:p-5">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p className="text-sm font-medium text-slate-600">预算</p>
            <h2 className="mt-1 text-2xl font-semibold">月度分类预算</h2>
          </div>
          <div className="flex flex-col gap-2 sm:flex-row">
            <input
              type="month"
              value={month}
              onChange={(event) => onMonthChange(event.target.value)}
              className="rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-emerald-700"
            />
            {member.role === "admin" ? (
              <button
                type="button"
                onClick={handleCopyPrevious}
                className="rounded-md border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700"
              >
                复制上月预算
              </button>
            ) : null}
          </div>
        </div>

        {loading ? <p className="mt-5 text-sm text-slate-500">正在加载...</p> : null}
        {error ? <p className="mt-5 text-sm text-red-600">{error}</p> : null}
        {submitError ? <p className="mt-5 text-sm text-red-600">{submitError}</p> : null}
        {message ? <p className="mt-5 text-sm text-emerald-700">{message}</p> : null}

        <div className="mt-5 space-y-3">
          {summary?.items.map((item) => (
            <div key={item.category.id} className="rounded-md border border-slate-200 p-3">
              <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                <div className="min-w-0">
                  <p className="text-sm font-medium">{item.category.name}</p>
                  <p className="mt-1 text-xs text-slate-500">
                    已花 {formatMoney(item.spentCents)} · {budgetStatusLabel(item.status)}
                  </p>
                </div>
                <div className="grid gap-2 sm:grid-cols-[10rem_auto]">
                  <input
                    value={amounts[item.category.id] ?? ""}
                    onChange={(event) => setAmounts((current) => ({ ...current, [item.category.id]: event.target.value }))}
                    disabled={member.role !== "admin"}
                    inputMode="decimal"
                    placeholder="预算金额"
                    className="rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-emerald-700 disabled:bg-slate-50"
                  />
                  {member.role === "admin" ? (
                    <button
                      type="button"
                      onClick={() => handleSave(item)}
                      className="rounded-md bg-emerald-700 px-3 py-2 text-sm font-medium text-white"
                    >
                      保存
                    </button>
                  ) : null}
                </div>
              </div>
              <div className="mt-3 h-2 rounded-full bg-slate-100">
                <div
                  className={`h-2 rounded-full ${budgetStatusBarClass(item.status)}`}
                  style={{ width: `${Math.min(item.usagePercent, 100)}%` }}
                />
              </div>
              <div className="mt-2 grid grid-cols-3 gap-2 text-xs text-slate-500">
                <span>预算 {item.budgetCents > 0 ? formatMoney(item.budgetCents) : "未设置"}</span>
                <span>剩余 {item.budgetCents > 0 ? formatMoney(item.remainingCents) : "-"}</span>
                <span>{item.budgetCents > 0 ? `${item.usagePercent}%` : "0%"}</span>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm md:p-5">
        <p className="text-sm font-medium text-slate-600">本月汇总</p>
        <div className="mt-4 grid gap-2">
          <Metric label="总预算" value={formatMoney(summary?.totalBudgetCents ?? 0)} />
          <Metric label="总支出" value={formatMoney(summary?.totalSpentCents ?? 0)} />
          <Metric label="剩余" value={formatMoney(summary?.totalRemainingCents ?? 0)} />
          <Metric label="接近预算" value={`${summary?.nearCount ?? 0} 个分类`} />
          <Metric label="已超支" value={`${summary?.overCount ?? 0} 个分类`} />
        </div>
      </section>
    </div>
  );
}

function SettingsPanel({
  member,
  categories,
  onCategoriesChanged
}: {
  member: Member;
  categories: Category[];
  onCategoriesChanged: () => Promise<void>;
}) {
  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_22rem]">
      <div className="space-y-4">
        <ChangePINPanel />
        {member.role === "admin" ? (
          <>
            <MembersPanel currentMemberId={member.id} />
            <CategoriesPanel categories={categories} onChanged={onCategoriesChanged} />
          </>
        ) : (
          <Panel eyebrow="设置" title="个人设置" body="当前账号是普通成员，可以修改自己的 PIN。成员和分类由管理员维护。" />
        )}
      </div>
      <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm md:p-5">
        <p className="text-sm font-medium text-slate-600">当前成员</p>
        <p className="mt-2 text-2xl font-semibold">{member.name}</p>
        <p className="mt-2 text-sm text-slate-500">{member.role === "admin" ? "管理员" : "普通成员"}</p>
      </div>
    </div>
  );
}

function ChangePINPanel() {
  const [currentPin, setCurrentPin] = useState("");
  const [newPin, setNewPin] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setMessage("");
    setSubmitting(true);
    try {
      await changeOwnPIN({ currentPin, newPin });
      setCurrentPin("");
      setNewPin("");
      setMessage("PIN 已更新");
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "修改 PIN 失败");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm md:p-5">
      <p className="text-sm font-medium text-slate-600">安全</p>
      <h2 className="mt-1 text-2xl font-semibold">修改 PIN</h2>
      <div className="grid gap-x-3 md:grid-cols-2">
        <FormField label="当前 PIN" value={currentPin} onChange={setCurrentPin} type="password" autoComplete="current-password" />
        <FormField label="新 PIN" value={newPin} onChange={setNewPin} type="password" autoComplete="new-password" />
      </div>
      {error ? <p className="mt-4 text-sm text-red-600">{error}</p> : null}
      {message ? <p className="mt-4 text-sm text-emerald-700">{message}</p> : null}
      <button
        type="submit"
        disabled={submitting}
        className="mt-5 rounded-md bg-emerald-700 px-4 py-3 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-60"
      >
        {submitting ? "正在保存" : "保存 PIN"}
      </button>
    </form>
  );
}

function MembersPanel({ currentMemberId }: { currentMemberId: string }) {
  const [members, setMembers] = useState<MemberAdmin[]>([]);
  const [name, setName] = useState("");
  const [role, setRole] = useState<"admin" | "member">("member");
  const [pin, setPin] = useState("");
  const [resetPins, setResetPins] = useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  async function refreshMembers() {
    const result = await listMembers();
    setMembers(result.members);
  }

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    listMembers()
      .then((result) => {
        if (!cancelled) {
          setMembers(result.members);
        }
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

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setMessage("");
    setSubmitting(true);
    try {
      const input: CreateMemberInput = { name, role, pin };
      await createMember(input);
      setName("");
      setRole("member");
      setPin("");
      await refreshMembers();
      setMessage("成员已创建");
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "创建成员失败");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDisable(id: string) {
    setError("");
    setMessage("");
    try {
      await disableMember(id);
      await refreshMembers();
      setMessage("成员已停用");
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "停用成员失败");
    }
  }

  async function handleResetPIN(id: string) {
    setError("");
    setMessage("");
    try {
      await resetMemberPIN(id, resetPins[id] ?? "");
      setResetPins((current) => ({ ...current, [id]: "" }));
      setMessage("PIN 已重置");
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "重置 PIN 失败");
    }
  }

  return (
    <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm md:p-5">
      <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
        <div>
          <p className="text-sm font-medium text-slate-600">成员</p>
          <h2 className="mt-1 text-2xl font-semibold">成员管理</h2>
        </div>
        {loading ? <p className="text-sm text-slate-500">加载中...</p> : null}
      </div>

      <form onSubmit={handleCreate} className="mt-4 grid gap-3 lg:grid-cols-[minmax(0,1fr)_9rem_9rem_auto] lg:items-end">
        <CompactField label="姓名" value={name} onChange={setName} />
        <label className="block">
          <span className="text-sm font-medium text-slate-700">角色</span>
          <select
            value={role}
            onChange={(event) => setRole(event.target.value as "admin" | "member")}
            className="mt-2 w-full rounded-md border border-slate-300 bg-white px-3 py-3 text-base outline-none focus:border-emerald-700"
          >
            <option value="member">成员</option>
            <option value="admin">管理员</option>
          </select>
        </label>
        <CompactField label="初始 PIN" value={pin} onChange={setPin} type="password" />
        <button
          type="submit"
          disabled={submitting}
          className="rounded-md bg-emerald-700 px-4 py-3 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-60"
        >
          新增
        </button>
      </form>

      <div className="mt-5 divide-y divide-slate-100">
        {members.map((item) => (
          <div key={item.id} className="grid gap-3 py-4 lg:grid-cols-[minmax(0,1fr)_8rem_12rem_auto] lg:items-center">
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{item.name}</p>
              <p className="mt-1 text-xs text-slate-500">
                {item.role === "admin" ? "管理员" : "成员"} · {item.active ? "启用" : "已停用"}
              </p>
            </div>
            <input
              value={resetPins[item.id] ?? ""}
              onChange={(event) => setResetPins((current) => ({ ...current, [item.id]: event.target.value }))}
              type="password"
              placeholder="新 PIN"
              className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-emerald-700"
            />
            <button
              type="button"
              onClick={() => handleResetPIN(item.id)}
              disabled={!item.active}
              className="rounded-md border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              重置 PIN
            </button>
            <button
              type="button"
              onClick={() => handleDisable(item.id)}
              disabled={!item.active || item.id === currentMemberId}
              className="rounded-md border border-red-200 px-3 py-2 text-sm font-medium text-red-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              停用
            </button>
          </div>
        ))}
      </div>
      {error ? <p className="mt-4 text-sm text-red-600">{error}</p> : null}
      {message ? <p className="mt-4 text-sm text-emerald-700">{message}</p> : null}
    </section>
  );
}

function CategoriesPanel({ categories, onChanged }: { categories: Category[]; onChanged: () => Promise<void> }) {
  const [name, setName] = useState("");
  const [type, setType] = useState<"expense" | "income">("expense");
  const [iconKey, setIconKey] = useState("tag");
  const [colorKey, setColorKey] = useState("emerald");
  const [edits, setEdits] = useState<Record<string, { name: string; iconKey: string; colorKey: string; sortOrder: string }>>({});
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    setEdits((current) => {
      const next = { ...current };
      for (const category of categories) {
        if (!next[category.id]) {
          next[category.id] = {
            name: category.name,
            iconKey: category.iconKey,
            colorKey: category.colorKey,
            sortOrder: String(category.sortOrder)
          };
        }
      }
      return next;
    });
  }, [categories]);

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setMessage("");
    setSubmitting(true);
    try {
      const input: CreateCategoryInput = { name, type, iconKey, colorKey };
      await createCategory(input);
      setName("");
      await onChanged();
      setMessage("分类已创建");
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "创建分类失败");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleUpdate(category: Category) {
    const edit = edits[category.id];
    if (!edit) {
      return;
    }
    setError("");
    setMessage("");
    try {
      await updateCategory(category.id, {
        name: edit.name,
        iconKey: edit.iconKey,
        colorKey: edit.colorKey,
        sortOrder: Number(edit.sortOrder) || 0
      });
      await onChanged();
      setMessage("分类已保存");
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "保存分类失败");
    }
  }

  async function handleDisable(id: string) {
    setError("");
    setMessage("");
    try {
      await disableCategory(id);
      await onChanged();
      setMessage("分类已停用");
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "停用分类失败");
    }
  }

  return (
    <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm md:p-5">
      <p className="text-sm font-medium text-slate-600">分类</p>
      <h2 className="mt-1 text-2xl font-semibold">分类管理</h2>

      <form onSubmit={handleCreate} className="mt-4 grid gap-3 lg:grid-cols-[minmax(0,1fr)_8rem_8rem_8rem_auto] lg:items-end">
        <CompactField label="名称" value={name} onChange={setName} />
        <label className="block">
          <span className="text-sm font-medium text-slate-700">类型</span>
          <select
            value={type}
            onChange={(event) => setType(event.target.value as "expense" | "income")}
            className="mt-2 w-full rounded-md border border-slate-300 bg-white px-3 py-3 text-base outline-none focus:border-emerald-700"
          >
            <option value="expense">支出</option>
            <option value="income">收入</option>
          </select>
        </label>
        <CompactField label="图标键" value={iconKey} onChange={setIconKey} />
        <CompactField label="颜色键" value={colorKey} onChange={setColorKey} />
        <button
          type="submit"
          disabled={submitting}
          className="rounded-md bg-emerald-700 px-4 py-3 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-60"
        >
          新增
        </button>
      </form>

      <div className="mt-5 divide-y divide-slate-100">
        {categories.map((category) => {
          const edit = edits[category.id] ?? {
            name: category.name,
            iconKey: category.iconKey,
            colorKey: category.colorKey,
            sortOrder: String(category.sortOrder)
          };
          return (
            <div key={category.id} className="grid gap-3 py-4 xl:grid-cols-[minmax(0,1fr)_8rem_8rem_6rem_auto_auto] xl:items-center">
              <input
                value={edit.name}
                onChange={(event) => setEdits((current) => ({ ...current, [category.id]: { ...edit, name: event.target.value } }))}
                className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-emerald-700"
              />
              <input
                value={edit.iconKey}
                onChange={(event) => setEdits((current) => ({ ...current, [category.id]: { ...edit, iconKey: event.target.value } }))}
                className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-emerald-700"
              />
              <input
                value={edit.colorKey}
                onChange={(event) => setEdits((current) => ({ ...current, [category.id]: { ...edit, colorKey: event.target.value } }))}
                className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-emerald-700"
              />
              <input
                value={edit.sortOrder}
                onChange={(event) => setEdits((current) => ({ ...current, [category.id]: { ...edit, sortOrder: event.target.value } }))}
                inputMode="numeric"
                className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-emerald-700"
              />
              <button
                type="button"
                onClick={() => handleUpdate(category)}
                className="rounded-md border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700"
              >
                保存
              </button>
              <button
                type="button"
                onClick={() => handleDisable(category.id)}
                className="rounded-md border border-red-200 px-3 py-2 text-sm font-medium text-red-700"
              >
                停用
              </button>
              <p className="text-xs text-slate-500 xl:col-span-6">{category.type === "expense" ? "支出" : "收入"}</p>
            </div>
          );
        })}
      </div>
      {error ? <p className="mt-4 text-sm text-red-600">{error}</p> : null}
      {message ? <p className="mt-4 text-sm text-emerald-700">{message}</p> : null}
    </section>
  );
}

function CompactField({
  label,
  value,
  onChange,
  type = "text"
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
}) {
  return (
    <label className="block">
      <span className="text-sm font-medium text-slate-700">{label}</span>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        type={type}
        className="mt-2 w-full rounded-md border border-slate-300 px-3 py-3 text-base outline-none focus:border-emerald-700"
      />
    </label>
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

function centsToYuanInput(cents: number): string {
  return (cents / 100).toFixed(2);
}

function formatMoney(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}

function budgetStatusLabel(status: string): string {
  if (status === "normal") {
    return "正常";
  }
  if (status === "near") {
    return "接近预算";
  }
  if (status === "over") {
    return "已超支";
  }
  return "未设置";
}

function budgetStatusBarClass(status: string): string {
  if (status === "over") {
    return "bg-red-600";
  }
  if (status === "near") {
    return "bg-amber-500";
  }
  if (status === "normal") {
    return "bg-emerald-700";
  }
  return "bg-slate-200";
}
