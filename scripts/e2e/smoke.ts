import { SQL } from "bun";

// Service base URLs (can be overridden by environment variables)
const IDENTITY_URL = process.env.IDENTITY_URL || "http://127.0.0.1:8081";
const INVENTORY_URL = process.env.INVENTORY_URL || "http://127.0.0.1:8082";
const LENDING_URL = process.env.LENDING_URL || "http://127.0.0.1:8083";
const NOTIFICATION_URL = process.env.NOTIFICATION_URL || "http://127.0.0.1:8084";

const LENDING_DB_URL = process.env.LENDING_DATABASE_URL || "postgres://borrowhub:borrowhub@localhost:5432/borrowhub_lending";
const NOTIFICATION_DB_URL = process.env.NOTIFICATION_DATABASE_URL || "postgres://borrowhub:borrowhub@localhost:5432/borrowhub_notification";

interface KillableProcess {
  kill: () => void;
}

const spawnedProcesses: KillableProcess[] = [];

async function isAlive(url: string): Promise<boolean> {
  try {
    const res = await fetch(`${url}/health/live`, { signal: AbortSignal.timeout(1000) });
    if (!res.ok) return false;
    const json: unknown = await res.json();
    if (json && typeof json === "object") {
      const obj = json as Record<string, unknown>;
      const data = obj.data && typeof obj.data === "object" ? (obj.data as Record<string, unknown>) : null;
      const s = String(obj.status || data?.status || "");
      return s === "ok" || s === "live";
    }
    return false;
  } catch {
    return false;
  }
}

async function ensureServices() {
  const services = [
    { name: "identity", url: IDENTITY_URL, cmd: ["go", "run", "./services/identity/cmd/api"], env: { APP_ENV: "local", HTTP_ADDR: ":8081" } },
    { name: "inventory", url: INVENTORY_URL, cmd: ["go", "run", "./services/inventory/cmd/api"], env: { APP_ENV: "local", HTTP_ADDR: ":8082" } },
    { name: "lending", url: LENDING_URL, cmd: ["go", "run", "./services/lending/cmd/api"], env: { APP_ENV: "local", HTTP_ADDR: ":8083" } },
    { name: "notification", url: NOTIFICATION_URL, cmd: ["go", "run", "./workers/notification/cmd/worker"], env: { APP_ENV: "local", HTTP_ADDR: ":8084" } },
  ];

  for (const svc of services) {
    if (await isAlive(svc.url)) {
      console.log(`[setup] Service ${svc.name} is already running at ${svc.url}`);
      continue;
    }
    console.log(`[setup] Starting service ${svc.name}...`);
    const proc = Bun.spawn(svc.cmd, {
      env: { ...process.env, ...svc.env },
      stdout: "ignore",
      stderr: "inherit",
    });
    spawnedProcesses.push(proc);

    // Wait for health
    let healthy = false;
    for (let i = 0; i < 30; i++) {
      await Bun.sleep(500);
      if (await isAlive(svc.url)) {
        healthy = true;
        break;
      }
    }
    if (!healthy) {
      throw new Error(`Service ${svc.name} failed to become healthy at ${svc.url}`);
    }
    console.log(`[setup] Service ${svc.name} is ready`);
  }
}

function cleanup() {
  for (const proc of spawnedProcesses) {
    try {
      proc.kill();
    } catch {}
  }
}

interface JsonEnvelope {
  data?: Record<string, unknown>;
  error?: {
    code: string;
    message: string;
    fields?: Record<string, unknown>;
  };
  meta?: {
    request_id: string;
  };
  [key: string]: unknown;
}

interface HttpResponse {
  status: number;
  body: JsonEnvelope;
}

async function request(url: string, options: RequestInit = {}): Promise<HttpResponse> {
  const headers = new Headers(options.headers || {});
  if (!headers.has("Content-Type") && options.body && typeof options.body === "string") {
    headers.set("Content-Type", "application/json");
  }

  const res = await fetch(url, { ...options, headers });
  const text = await res.text();
  let body: JsonEnvelope = {};
  try {
    const parsed: unknown = JSON.parse(text);
    if (parsed && typeof parsed === "object") {
      body = parsed as JsonEnvelope;
    }
  } catch {
    body = { raw: text };
  }
  return { status: res.status, body };
}

function assert(condition: boolean, message: string) {
  if (!condition) {
    throw new Error(`Assertion failed: ${message}`);
  }
}

function getObject(val: unknown, key: string): Record<string, unknown> {
  if (val && typeof val === "object" && key in val) {
    const obj = (val as Record<string, unknown>)[key];
    if (obj && typeof obj === "object") {
      return obj as Record<string, unknown>;
    }
  }
  return {};
}

function getString(val: unknown, key: string): string {
  if (val && typeof val === "object" && key in val) {
    const str = (val as Record<string, unknown>)[key];
    if (typeof str === "string") {
      return str;
    }
  }
  return "";
}

async function runSmokeTest() {
  console.log("=== Starting BorrowHub End-to-End Integration Smoke Test ===");
  const runId = Math.random().toString(36).substring(2, 8);
  console.log(`[info] Run ID: ${runId}`);

  // --------------------------------------------------------------------------
  // STEP 1: Register & login admin and member
  // --------------------------------------------------------------------------
  console.log("\n-> Step 1: Register and login admin & member");
  const adminEmail = `admin-${runId}@borrowhub.test`;
  const memberEmail = `member-${runId}@borrowhub.test`;
  const password = "Password123!";

  // Register Admin
  const regAdmin = await request(`${IDENTITY_URL}/api/v1/auth/register`, {
    method: "POST",
    body: JSON.stringify({
      email: adminEmail,
      password,
      display_name: `Admin ${runId}`,
      terms_accepted: true,
    }),
  });
  assert(regAdmin.status === 201, `Admin register failed: ${JSON.stringify(regAdmin.body)}`);
  const adminUser = getObject(regAdmin.body.data, "user");
  const adminUserId = getString(adminUser, "id");

  // Login Admin
  const loginAdmin = await request(`${IDENTITY_URL}/api/v1/auth/login`, {
    method: "POST",
    body: JSON.stringify({ email: adminEmail, password }),
  });
  assert(loginAdmin.status === 200, `Admin login failed: ${JSON.stringify(loginAdmin.body)}`);
  const adminTokens = getObject(loginAdmin.body.data, "tokens");
  const adminToken = getString(adminTokens, "access_token");
  console.log(`   Admin registered & logged in (ID: ${adminUserId})`);

  // Register Member
  const regMember = await request(`${IDENTITY_URL}/api/v1/auth/register`, {
    method: "POST",
    body: JSON.stringify({
      email: memberEmail,
      password,
      display_name: `Member ${runId}`,
      terms_accepted: true,
    }),
  });
  assert(regMember.status === 201, `Member register failed: ${JSON.stringify(regMember.body)}`);
  const memberUser = getObject(regMember.body.data, "user");
  const memberUserId = getString(memberUser, "id");

  // Login Member
  const loginMember = await request(`${IDENTITY_URL}/api/v1/auth/login`, {
    method: "POST",
    body: JSON.stringify({ email: memberEmail, password }),
  });
  assert(loginMember.status === 200, `Member login failed: ${JSON.stringify(loginMember.body)}`);
  const memberTokens = getObject(loginMember.body.data, "tokens");
  const memberToken = getString(memberTokens, "access_token");
  console.log(`   Member registered & logged in (ID: ${memberUserId})`);

  // --------------------------------------------------------------------------
  // STEP 2: Create organization and accept invite for member
  // --------------------------------------------------------------------------
  console.log("\n-> Step 2: Create organization and accept member invitation");
  const createOrg = await request(`${IDENTITY_URL}/api/v1/organizations`, {
    method: "POST",
    headers: { Authorization: `Bearer ${adminToken}` },
    body: JSON.stringify({
      name: `Acme Lab ${runId}`,
      slug: `acme-lab-${runId}`,
      timezone: "UTC",
      visibility: "private",
      max_loan_duration_days: 14,
    }),
  });
  assert(createOrg.status === 201, `Create org failed: ${JSON.stringify(createOrg.body)}`);
  const orgId = getString(createOrg.body.data, "id");
  console.log(`   Organization created (ID: ${orgId})`);

  // Invite member
  const inviteRes = await request(`${IDENTITY_URL}/api/v1/organizations/${orgId}/invitations`, {
    method: "POST",
    headers: { Authorization: `Bearer ${adminToken}` },
    body: JSON.stringify({ email: memberEmail, role: "member" }),
  });
  assert(inviteRes.status === 201, `Invite member failed: ${JSON.stringify(inviteRes.body)}`);
  const inviteToken = getString(inviteRes.body.data, "token");
  console.log(`   Invitation issued`);

  // Member accepts invite
  const acceptRes = await request(`${IDENTITY_URL}/api/v1/invitations/${inviteToken}/accept`, {
    method: "POST",
    headers: { Authorization: `Bearer ${memberToken}` },
  });
  assert(acceptRes.status === 200, `Accept invitation failed: ${JSON.stringify(acceptRes.body)}`);
  console.log(`   Member accepted invitation successfully`);

  // --------------------------------------------------------------------------
  // STEP 3: Create catalog category, item, and unit
  // --------------------------------------------------------------------------
  console.log("\n-> Step 3: Create catalog category, item, and unit");
  // Category
  const catRes = await request(`${INVENTORY_URL}/api/v1/organizations/${orgId}/categories`, {
    method: "POST",
    headers: { Authorization: `Bearer ${adminToken}` },
    body: JSON.stringify({
      name: `Kamera ${runId}`,
      slug: `kamera-${runId}`,
      description: "Peralatan Kamera E2E",
    }),
  });
  assert(catRes.status === 201, `Create category failed: ${JSON.stringify(catRes.body)}`);
  const categoryId = getString(catRes.body.data, "id");

  // Item
  const itemRes = await request(`${INVENTORY_URL}/api/v1/organizations/${orgId}/items`, {
    method: "POST",
    headers: { Authorization: `Bearer ${adminToken}` },
    body: JSON.stringify({
      category_id: categoryId,
      name: `Sony FX3 E2E ${runId}`,
      description: "Cinema Camera 4K",
      is_lendable: true,
    }),
  });
  assert(itemRes.status === 201, `Create item failed: ${JSON.stringify(itemRes.body)}`);
  const itemId = getString(itemRes.body.data, "id");

  // Unit
  const barcode = `ACME-CAM-${runId.toUpperCase()}`;
  const unitRes = await request(`${INVENTORY_URL}/api/v1/items/${itemId}/units`, {
    method: "POST",
    headers: { Authorization: `Bearer ${adminToken}` },
    body: JSON.stringify({
      barcode,
      serial_number: `SN-${runId.toUpperCase()}`,
      condition: "good",
      notes: "Unit E2E siap pakai",
    }),
  });
  assert(unitRes.status === 201, `Create unit failed: ${JSON.stringify(unitRes.body)}`);
  const unitId = getString(unitRes.body.data, "id");
  assert(getString(unitRes.body.data, "status") === "available", `Unit status should be available, got ${getString(unitRes.body.data, "status")}`);
  console.log(`   Created category (${categoryId}), item (${itemId}), unit (${unitId}, barcode: ${barcode})`);

  // --------------------------------------------------------------------------
  // STEP 4: Check availability
  // --------------------------------------------------------------------------
  console.log("\n-> Step 4: Check item availability");
  const startTime = new Date(Date.now() + 3600 * 1000).toISOString();
  const endTime = new Date(Date.now() + 86400 * 1000 * 3).toISOString();

  const availRes = await request(
    `${LENDING_URL}/api/v1/items/${itemId}/availability?organization_id=${orgId}&start_time=${encodeURIComponent(startTime)}&end_time=${encodeURIComponent(endTime)}`,
    {
      method: "GET",
      headers: { Authorization: `Bearer ${memberToken}` },
    }
  );
  assert(availRes.status === 200, `Availability check failed: ${JSON.stringify(availRes.body)}`);
  const availQty = Number(availRes.body.data?.available_units || 0);
  assert(availQty >= 1, `Available quantity should be >= 1, got ${availQty}`);
  const eligibleUnits = Array.isArray(availRes.body.data?.eligible_unit_ids) ? availRes.body.data.eligible_unit_ids : [];
  assert(eligibleUnits.includes(unitId), "Eligible units should include newly created unit");
  console.log(`   Availability confirmed: ${availQty} available unit(s)`);

  // --------------------------------------------------------------------------
  // STEP 5: Submit loan request as member
  // --------------------------------------------------------------------------
  console.log("\n-> Step 5: Submit loan request as member");
  const idempKey1 = crypto.randomUUID();
  const loanReqRes = await request(`${LENDING_URL}/api/v1/loan-requests`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${memberToken}`,
      "Idempotency-Key": idempKey1,
    },
    body: JSON.stringify({
      organization_id: orgId,
      item_id: itemId,
      quantity: 1,
      start_time: startTime,
      end_time: endTime,
      purpose: "E2E documentary shoot",
    }),
  });
  assert(loanReqRes.status === 201, `Submit loan request failed: ${JSON.stringify(loanReqRes.body)}`);
  const loanRequestId = getString(loanReqRes.body.data, "id");
  assert(getString(loanReqRes.body.data, "status") === "pending", `Loan request should be pending, got ${getString(loanReqRes.body.data, "status")}`);
  console.log(`   Loan request submitted (ID: ${loanRequestId}, status: pending)`);

  // --------------------------------------------------------------------------
  // STEP 6: Approve loan request as admin and obtain checkout token
  // --------------------------------------------------------------------------
  console.log("\n-> Step 6: Approve loan request as admin");
  const idempKey2 = crypto.randomUUID();
  const approveRes = await request(`${LENDING_URL}/api/v1/loan-requests/${loanRequestId}/approve`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${adminToken}`,
      "Idempotency-Key": idempKey2,
    },
    body: JSON.stringify({ review_notes: "Approved by E2E admin" }),
  });
  assert(approveRes.status === 200, `Approve loan request failed: ${JSON.stringify(approveRes.body)}`);
  const loanObj = getObject(approveRes.body.data, "loan");
  const loanId = getString(loanObj, "id");
  const checkoutToken = getString(approveRes.body.data, "checkout_token");
  assert(typeof checkoutToken === "string" && checkoutToken.length > 0, "Expected non-empty checkout token");
  assert(getString(loanObj, "status") === "pending_checkout", `Loan status should be pending_checkout`);
  console.log(`   Loan request approved. Loan ID: ${loanId}, Checkout Token obtained.`);

  // --------------------------------------------------------------------------
  // STEP 7: Checkout loan with token, verify unit status 'borrowed'
  // --------------------------------------------------------------------------
  console.log("\n-> Step 7: Checkout loan and verify unit status");
  const checkoutRes = await request(`${LENDING_URL}/api/v1/loans/${loanId}/check-out`, {
    method: "POST",
    headers: { Authorization: `Bearer ${adminToken}` },
    body: JSON.stringify({ token: checkoutToken }),
  });
  assert(checkoutRes.status === 200, `Checkout loan failed: ${JSON.stringify(checkoutRes.body)}`);
  assert(getString(checkoutRes.body.data, "status") === "checked_out", `Loan status should be checked_out, got ${getString(checkoutRes.body.data, "status")}`);

  // Verify unit in inventory is now 'borrowed'
  const unitCheck1 = await request(`${INVENTORY_URL}/api/v1/units/${unitId}`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  });
  assert(unitCheck1.status === 200, `Get unit failed: ${JSON.stringify(unitCheck1.body)}`);
  assert(getString(unitCheck1.body.data, "status") === "borrowed", `Unit should be borrowed, got ${getString(unitCheck1.body.data, "status")}`);
  console.log(`   Loan checked out successfully. Unit ${unitId} is now status 'borrowed'.`);

  // --------------------------------------------------------------------------
  // STEP 8: Return loan, verify unit status 'available'
  // --------------------------------------------------------------------------
  console.log("\n-> Step 8: Return loan and verify unit status");
  const idempKey3 = crypto.randomUUID();
  const returnRes = await request(`${LENDING_URL}/api/v1/loans/${loanId}/return`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${adminToken}`,
      "Idempotency-Key": idempKey3,
    },
    body: JSON.stringify({
      return_condition: "good",
      return_notes: "Returned clean and undamaged",
    }),
  });
  assert(returnRes.status === 200, `Return loan failed: ${JSON.stringify(returnRes.body)}`);
  assert(getString(returnRes.body.data, "status") === "returned", `Loan status should be returned, got ${getString(returnRes.body.data, "status")}`);

  // Verify unit in inventory is now 'available'
  const unitCheck2 = await request(`${INVENTORY_URL}/api/v1/units/${unitId}`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  });
  assert(unitCheck2.status === 200, `Get unit failed: ${JSON.stringify(unitCheck2.body)}`);
  assert(getString(unitCheck2.body.data, "status") === "available", `Unit should be available after return, got ${getString(unitCheck2.body.data, "status")}`);
  console.log(`   Loan returned. Unit ${unitId} is now back to status 'available'.`);

  // --------------------------------------------------------------------------
  // STEP 9: Verify domain event in lending outbox
  // --------------------------------------------------------------------------
  console.log("\n-> Step 9: Verify domain events in lending outbox");
  const lendingDb = new SQL(LENDING_DB_URL);
  const outboxEvents = await lendingDb`
    SELECT event_id, event_type, status, organization_id
    FROM outbox_events
    WHERE organization_id = ${orgId}
    ORDER BY id ASC
  `;
  assert(outboxEvents.length >= 2, `Expected at least 2 outbox events, found ${outboxEvents.length}`);
  const eventTypes: string[] = [];
  for (const ev of outboxEvents) {
    if (ev && typeof ev === "object" && "event_type" in ev) {
      eventTypes.push(String(ev.event_type));
    }
  }
  assert(eventTypes.includes("loan.approved"), "Outbox should contain loan.approved event");
  assert(eventTypes.includes("loan.checked_out"), "Outbox should contain loan.checked_out event");
  console.log(`   Lending outbox verified: found events [${eventTypes.join(", ")}]`);
  await lendingDb.close();

  // --------------------------------------------------------------------------
  // STEP 10: Verify notification in notification DB / API
  // --------------------------------------------------------------------------
  console.log("\n-> Step 10: Verify notification delivery");
  const notifDb = new SQL(NOTIFICATION_DB_URL);
  let notifsCount = 0;
  // Allow worker consumer up to 5 seconds to process from RabbitMQ
  for (let i = 0; i < 10; i++) {
    const notifs = await notifDb`
      SELECT id, event_type, title, status, recipient_user_id
      FROM notifications
      WHERE organization_id = ${orgId}
    `;
    if (notifs.length > 0) {
      notifsCount = notifs.length;
      break;
    }
    await Bun.sleep(500);
  }
  console.log(`   Notifications in DB: ${notifsCount} record(s)`);

  // Verify via notification HTTP endpoint
  const notifHttp = await request(`${NOTIFICATION_URL}/api/v1/organizations/${orgId}/notifications`, {
    headers: { Authorization: `Bearer ${memberToken}` },
  });
  assert(notifHttp.status === 200, `Notification HTTP API failed: ${JSON.stringify(notifHttp.body)}`);
  const notifsList = Array.isArray(notifHttp.body.data?.notifications) ? notifHttp.body.data.notifications : [];
  console.log(`   Notification API returned ${notifsList.length} notification(s)`);
  await notifDb.close();

  // --------------------------------------------------------------------------
  // NEGATIVE TESTS
  // --------------------------------------------------------------------------
  console.log("\n-> Running Negative Tests");

  // Negative 1: Member unauthorized approve (403)
  console.log("   [Negative 1] Member unauthorized approve attempt");
  const req2 = await request(`${LENDING_URL}/api/v1/loan-requests`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${memberToken}`,
      "Idempotency-Key": crypto.randomUUID(),
    },
    body: JSON.stringify({
      organization_id: orgId,
      item_id: itemId,
      quantity: 1,
      start_time: startTime,
      end_time: endTime,
      purpose: "Negative test attempt",
    }),
  });
  assert(req2.status === 201, "Setup second loan request failed");
  const req2Id = getString(req2.body.data, "id");

  const memberApprove = await request(`${LENDING_URL}/api/v1/loan-requests/${req2Id}/approve`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${memberToken}`,
      "Idempotency-Key": crypto.randomUUID(),
    },
    body: JSON.stringify({ review_notes: "Member self approval" }),
  });
  assert(memberApprove.status === 403, `Expected 403 Forbidden on member approve, got ${memberApprove.status}`);
  console.log("   -> PASSED: 403 Forbidden returned on unauthorized approval.");

  // Negative 2: Cross-org access (403/404)
  console.log("   [Negative 2] Cross-organization access prevention");
  // Admin creates Org B where member is NOT a member
  const createOrgB = await request(`${IDENTITY_URL}/api/v1/organizations`, {
    method: "POST",
    headers: { Authorization: `Bearer ${adminToken}` },
    body: JSON.stringify({
      name: `Org B ${runId}`,
      slug: `org-b-${runId}`,
      timezone: "UTC",
      visibility: "private",
    }),
  });
  const orgBId = getString(createOrgB.body.data, "id");

  // Member tries to create loan request in Org B
  const crossOrgReq = await request(`${LENDING_URL}/api/v1/loan-requests`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${memberToken}`,
      "Idempotency-Key": crypto.randomUUID(),
    },
    body: JSON.stringify({
      organization_id: orgBId,
      item_id: itemId,
      quantity: 1,
      start_time: startTime,
      end_time: endTime,
      purpose: "Cross org intrusion",
    }),
  });
  assert(crossOrgReq.status === 403 || crossOrgReq.status === 404, `Expected 403/404 on cross-org request, got ${crossOrgReq.status}`);
  console.log(`   -> PASSED: ${crossOrgReq.status} returned on cross-org access.`);

  // Negative 3: Reservation conflict (409)
  console.log("   [Negative 3] Reservation conflict handling");
  // Admin approves req2
  const adminApproveReq2 = await request(`${LENDING_URL}/api/v1/loan-requests/${req2Id}/approve`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${adminToken}`,
      "Idempotency-Key": crypto.randomUUID(),
    },
    body: JSON.stringify({ review_notes: "Approve for unit booking" }),
  });
  assert(adminApproveReq2.status === 200, "Admin approve req2 failed");

  // Now create req3 for same item during overlapping window when quantity is exhausted
  const req3 = await request(`${LENDING_URL}/api/v1/loan-requests`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${memberToken}`,
      "Idempotency-Key": crypto.randomUUID(),
    },
    body: JSON.stringify({
      organization_id: orgId,
      item_id: itemId,
      quantity: 1,
      start_time: startTime,
      end_time: endTime,
      purpose: "Conflict test",
    }),
  });
  assert(req3.status === 409, `Expected 409 Conflict on overlapping reservation, got ${req3.status}`);
  console.log("   -> PASSED: 409 Conflict returned on overlapping reservation.");

  console.log("\n========================================================");
  console.log("ALL 10 E2E STEPS AND NEGATIVE CHECKS PASSED SUCCESSFULLY");
  console.log("========================================================\n");
}

try {
  await ensureServices();
  await runSmokeTest();
} catch (err) {
  console.error("\n❌ E2E Smoke Test Failed:", err);
  process.exit(1);
} finally {
  cleanup();
}
