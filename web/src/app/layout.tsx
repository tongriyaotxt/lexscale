import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import './globals.css';

const inter = Inter({ subsets: ['latin'] });

export const metadata: Metadata = {
  title: '律衡 LexScale — 开源法律 AI Agent 框架',
  description: '双图模型 + 三层证据 + 审批门 + 6 Agent 协作。Build Legal AI Agents in Minutes, Not Months.',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="zh-CN">
      <body className={inter.className}>
        <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
          {/* 导航栏 */}
          <nav className="bg-legal-900 dark:bg-legal-950 text-white shadow-lg">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
              <div className="flex justify-between h-16 items-center">
                <a href="/" className="flex items-center hover:opacity-90 transition">
                  <span className="text-2xl">⚖️</span>
                  <div className="ml-2">
                    <span className="text-lg font-bold">律衡 LexScale</span>
                    <span className="ml-2 text-sm text-legal-300 hidden sm:inline">权衡之间，法律智能</span>
                  </div>
                </a>
                <div className="flex space-x-1">
                  <NavLink href="/">案件</NavLink>
                  <NavLink href="/dashboard">仪表盘</NavLink>
                  <NavLink href="/approvals">审批</NavLink>
                </div>
              </div>
            </div>
          </nav>

          {/* 主内容 */}
          <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
            {children}
          </main>

          {/* 页脚 */}
          <footer className="border-t border-gray-200 dark:border-gray-800 mt-12 py-6">
            <div className="max-w-7xl mx-auto px-4 text-center text-sm text-gray-400">
              律衡 LexScale — Apache 2.0 License — ⚖️ Where Law Meets Intelligence
            </div>
          </footer>
        </div>
      </body>
    </html>
  );
}

function NavLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <a
      href={href}
      className="px-3 py-2 rounded-md text-sm font-medium text-legal-100 hover:bg-legal-800 hover:text-white transition"
    >
      {children}
    </a>
  );
}
