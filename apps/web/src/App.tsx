import * as Tabs from "@radix-ui/react-tabs";
import type { ReactNode } from "react";

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
        </aside>

        <section className="mx-auto flex min-h-screen w-full max-w-6xl flex-col px-4 pb-24 pt-5 md:px-8 md:pb-8 lg:px-10">
          <header className="md:hidden">
            <BrandHeader icon={icon} compact />
          </header>

          <div className="hidden items-start justify-between gap-6 md:flex">
            <div>
              <p className="text-sm font-medium text-emerald-700">Commune</p>
              <h1 className="mt-1 text-2xl font-semibold">家庭共享账本</h1>
            </div>
            <div className="rounded-lg border border-slate-200 bg-white px-4 py-3 text-sm text-slate-600 shadow-sm">
              当前阶段：基础框架
            </div>
          </div>

          <div className="mt-6 flex-1 md:mt-8">
            <Tabs.Content value="add" className="h-full">
              <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_20rem]">
                <Panel
                  eyebrow="下一步"
                  title="建立记账基础流程"
                  body="当前版本只验证前端壳、样式系统和构建链路。后续这里会成为快速记账表单。"
                />
                <Panel
                  eyebrow="本月概览"
                  title="预算和最近流水"
                  body="PC 端会把辅助信息放在右侧，移动端则在主内容下方顺序展示。"
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
