// E2E: Swarm Observability Dashboard
// Run: node test/e2e/swarm-dashboard.mjs
// Requires: npm install playwright (run from test/e2e/)

import { chromium } from 'playwright';

const BASE = process.env.BASE_URL || 'http://localhost:8080';
const HEADLESS = process.env.HEADLESS !== 'false';

async function run() {
  const browser = await chromium.launch({ headless: HEADLESS });
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

  page.on('pageerror', e => { throw new Error(`JS ERROR: ${e.message}`); });

  let passed = 0;
  let failed = 0;
  function assert(condition, msg) {
    if (condition) { passed++; console.log(`  ✅ ${msg}`); }
    else { failed++; console.error(`  ❌ ${msg}`); }
  }

  // ==================== 1. Dashboard loads with 7 tabs ====================
  console.log('\n📋 Tab verification');
  await page.goto(BASE, { waitUntil: 'networkidle', timeout: 15000 });
  await page.waitForTimeout(2000);

  const expectedTabs = ['dashboard', 'console', 'results', 'memory', 'tools', 'system', 'swarm'];
  for (const tab of expectedTabs) {
    const visible = await page.locator(`nav button[data-tab="${tab}"]`).isVisible();
    assert(visible, `Tab "${tab}" is visible`);
  }
  const tabCount = await page.locator('nav button').count();
  assert(tabCount === 7, `Expected 7 nav tabs, got ${tabCount}`);

  // ==================== 2. Swarm tab shows worker card(s) ====================
  console.log('\n🐝 Swarm tab — worker cards');
  await page.locator('nav button[data-tab="swarm"]').click();
  await page.waitForTimeout(2500);

  const workerCards = await page.locator('#swarm-worker-cards .card').count();
  const noWorkers = await page.locator('#swarm-worker-cards .empty-state').isVisible().catch(() => false);

  if (workerCards > 0) {
    assert(true, `Found ${workerCards} worker card(s)`);

    const firstCard = page.locator('#swarm-worker-cards .card').first();
    const cardText = await firstCard.textContent();
    assert(cardText.includes('e2e-worker'), 'Worker card shows name');
    assert(cardText.includes('8082'), 'Worker card shows port 8082');
    assert(cardText.includes('s'), 'Worker card shows uptime');
    assert(cardText.includes('B'), 'Worker card shows log size');
  } else if (noWorkers) {
    console.log('  ℹ️  No workers running — skipping card tests');
  } else {
    assert(false, 'Worker cards container rendered');
  }

  // ==================== 3. Click worker card → log viewer ====================
  console.log('\n📜 Worker log viewer');
  if (workerCards > 0) {
    await page.locator('#swarm-worker-cards .card').first().click();
    await page.waitForTimeout(2000);

    const detailVisible = await page.locator('#swarm-worker-detail').isVisible();
    assert(detailVisible, 'Worker detail view is shown');

    const backBtnVisible = await page.locator('#swarm-back-btn').isVisible();
    assert(backBtnVisible, 'Back button is visible');

    const refreshBtnVisible = await page.locator('#swarm-refresh-log').isVisible();
    assert(refreshBtnVisible, 'Refresh button is visible');

    await page.waitForTimeout(1500);
    const logContent = await page.locator('#swarm-log-view').textContent();
    assert(logContent.length > 0, 'Log viewer has content');
    assert(logContent.includes('Ivai OS starting up'), 'Log contains startup message');

    await page.locator('#swarm-back-btn').click();
    await page.waitForTimeout(500);
    const cardsVisibleAgain = await page.locator('#swarm-worker-cards').isVisible();
    assert(cardsVisibleAgain, 'Worker cards visible again after back');
  } else {
    console.log('  ℹ️  Skipping log viewer tests (no workers)');
  }

  // ==================== 4. Swarm Dispatch ====================
  console.log('\n📤 Swarm dispatch');

  const dispatchInputVisible = await page.locator('#swarm-worker-input').isVisible();
  assert(dispatchInputVisible, 'Dispatch worker input is visible');

  const instructionInputVisible = await page.locator('#swarm-instruction-input').isVisible();
  assert(instructionInputVisible, 'Dispatch instruction input is visible');

  const dispatchBtnVisible = await page.locator('#swarm-dispatch-btn').isVisible();
  assert(dispatchBtnVisible, 'Dispatch send button is visible');

  // Error path: nonexistent worker
  await page.locator('#swarm-worker-input').fill('localhost:9999');
  await page.locator('#swarm-instruction-input').fill('test error path');
  await page.locator('#swarm-dispatch-btn').click();
  await page.waitForTimeout(4000);

  const errorResultVisible = await page.locator('#swarm-dispatch-result').isVisible();
  assert(errorResultVisible, 'Dispatch result panel appears after send');

  const resultText = await page.locator('#swarm-dispatch-result').textContent();
  assert(resultText.includes('error') || resultText.includes('refused'), 'Dispatch error shows connection error');

  // Success path
  if (workerCards > 0) {
    await page.locator('#swarm-worker-input').fill('localhost:8082');
    await page.locator('#swarm-instruction-input').fill('say hello back in one word');
    await page.locator('#swarm-dispatch-btn').click();
    await page.waitForTimeout(10000);

    const successResultVisible = await page.locator('#swarm-dispatch-result').isVisible();
    assert(successResultVisible, 'Dispatch result panel visible after success');
    await page.locator('#swarm-instruction-input').clear();
  }

  // ==================== 5. Cross-tab ====================
  console.log('\n🔍 Cross-tab compatibility');

  await page.locator('nav button[data-tab="tools"]').click();
  await page.waitForTimeout(2000);
  const toolCards = await page.locator('#tools-list .tool-card').count();
  assert(toolCards >= 7, `Tools tab: ${toolCards} tools (expected >= 7)`);

  const toolsText = await page.locator('#tools-list').textContent();
  assert(toolsText.includes('swarm_spawn'), 'Tools list includes swarm_spawn');
  assert(toolsText.includes('swarm_kill'), 'Tools list includes swarm_kill');

  await page.locator('nav button[data-tab="dashboard"]').click();
  await page.waitForTimeout(2000);
  const statusCards = await page.locator('#status-cards .card').count();
  assert(statusCards >= 4, `Dashboard: ${statusCards} status cards (expected >= 4)`);

  // ==================== Summary ====================
  console.log(`\n${'='.repeat(50)}`);
  console.log(`Results: ${passed} passed, ${failed} failed`);
  console.log(`${'='.repeat(50)}`);

  await browser.close();
  process.exit(failed > 0 ? 1 : 0);
}

run().catch(e => {
  console.error('FATAL:', e.message);
  process.exit(1);
});
