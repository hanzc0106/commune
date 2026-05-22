import * as Tabs from "@radix-ui/react-tabs";
import type { ReactNode } from "react";

type AppProps = {
  icon: ReactNode;
};

export function App({ icon }: AppProps) {
  return (
    <main className="min-h-screen bg-stone-50 text-slate-950">
      <section className="mx-auto flex min-h-screen w-full max-w-md flex-col px-4 pb-6 pt-5">
        <header className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-700 text-white">
            {icon}
          </div>
          <div>
            <h1 className="text-xl font-semibold">Commune</h1>
            <p className="text-sm text-slate-600">家庭共享账本</p>
          </div>
        </header>

        <Tabs.Root defaultValue="add" className="mt-6 flex flex-1 flex-col">
          <Tabs.Content value="add" className="flex-1">
            <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
              <p className="text-sm font-medium text-slate-600">下一步</p>
              <p className="mt-2 text-2xl font-semibold">建立记账基础流程</p>
              <p className="mt-2 text-sm leading-6 text-slate-600">
                当前版本只验证前端壳、样式系统和构建链路。
              </p>
            </div>
          </Tabs.Content>

          <Tabs.List className="mt-6 grid grid-cols-4 rounded-lg border border-slate-200 bg-white p-1 text-sm shadow-sm">
            <Tabs.Trigger value="add" className="rounded-md px-2 py-2 data-[state=active]:bg-emerald-700 data-[state=active]:text-white">
              记一笔
            </Tabs.Trigger>
            <Tabs.Trigger value="transactions" className="rounded-md px-2 py-2 data-[state=active]:bg-emerald-700 data-[state=active]:text-white">
              流水
            </Tabs.Trigger>
            <Tabs.Trigger value="budgets" className="rounded-md px-2 py-2 data-[state=active]:bg-emerald-700 data-[state=active]:text-white">
              预算
            </Tabs.Trigger>
            <Tabs.Trigger value="settings" className="rounded-md px-2 py-2 data-[state=active]:bg-emerald-700 data-[state=active]:text-white">
              设置
            </Tabs.Trigger>
          </Tabs.List>
        </Tabs.Root>
      </section>
    </main>
  );
}
