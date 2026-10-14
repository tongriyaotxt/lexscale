// ─── E2E 测试：页面加载与导航 ────────────────────────────
import { test, expect } from '@playwright/test';

test.describe('律衡 LexScale — 页面加载', () => {
  test('首页加载并显示案件列表', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle(/律衡|LexScale/);
    await expect(page.locator('h1')).toContainText('案件管理');
  });

  test('仪表盘页面加载', async ({ page }) => {
    await page.goto('/dashboard');
    await expect(page.locator('h1')).toContainText('风险仪表盘');
    // 统计卡片
    await expect(page.locator('text=总案件')).toBeVisible();
    await expect(page.locator('text=风险发现')).toBeVisible();
    await expect(page.locator('text=待审批')).toBeVisible();
  });

  test('审批工作台加载', async ({ page }) => {
    await page.goto('/approvals');
    await expect(page.locator('h1')).toContainText('审批工作台');
    // 过滤按钮
    await expect(page.locator('text=待审批')).toBeVisible();
    await expect(page.locator('text=已批准')).toBeVisible();
  });

  test('导航栏链接工作', async ({ page }) => {
    await page.goto('/');
    // 点击仪表盘
    await page.click('a[href="/dashboard"]');
    await expect(page).toHaveURL(/.*dashboard/);
    // 点击审批
    await page.click('a[href="/approvals"]');
    await expect(page).toHaveURL(/.*approvals/);
    // 点击案件
    await page.click('a[href="/"]');
    await expect(page).toHaveURL(/.*\/$/);
  });
});

test.describe('律衡 LexScale — 案件操作', () => {
  test('创建新案件（演示模式）', async ({ page }) => {
    await page.goto('/');
    await page.click('text=新建案件');
    await expect(page.locator('text=案件名称')).toBeVisible();

    await page.fill('input[type="text"]', 'E2E 测试案件');
    await page.fill('textarea', '这是一个 E2E 自动测试创建的案件');
    await page.click('text=创建');

    // 验证新案件出现
    await expect(page.locator('text=E2E 测试案件')).toBeVisible();
  });

  test('案件详情页加载', async ({ page }) => {
    await page.goto('/');
    // 点击第一个案件的查看详情
    const detailLink = page.locator('text=查看详情').first();
    if (await detailLink.isVisible()) {
      await detailLink.click();
      // 应该跳转到 matters/[id]
      await expect(page).toHaveURL(/.*matters\/.+/);
      // Tab 应该存在
      await expect(page.locator('text=双图可视化')).toBeVisible();
      await expect(page.locator('text=风险发现')).toBeVisible();
      await expect(page.locator('text=实体列表')).toBeVisible();
    }
  });
});

test.describe('律衡 LexScale — 审批操作', () => {
  test('审批过滤切换', async ({ page }) => {
    await page.goto('/approvals');
    // 切换到全部
    await page.click('button:has-text("全部")');
    // 切换到已批准
    await page.click('button:has-text("已批准")');
    // 切换回待审批
    await page.click('button:has-text("待审批")');
  });

  test('批准一个待审批项', async ({ page }) => {
    await page.goto('/approvals');
    const approveBtn = page.locator('button:has-text("批准")').first();
    if (await approveBtn.isVisible()) {
      await approveBtn.click();
      // 状态应该变为已批准
      await expect(page.locator('text=已批准').first()).toBeVisible();
    }
  });
});

test.describe('律衡 LexScale — 响应式', () => {
  test('移动端视口适配', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('案件管理');
  });
});
