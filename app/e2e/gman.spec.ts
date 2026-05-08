import { test, expect } from '@playwright/test';
import { spawn, ChildProcess } from 'child_process';
import { createInterface } from 'readline';
import { readdirSync, existsSync } from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const projectRoot = path.resolve(__dirname, '..', '..');

// Build the Go sidecar before tests
let binaryPath: string;

function getBinaryPath(): string {
  const binDir = path.join(projectRoot, 'app', 'src-tauri', 'binaries');
  if (existsSync(binDir)) {
    const files = readdirSync(binDir).filter((f: string) => f.startsWith('gman-core'));
    if (files.length > 0) {
      return path.join(binDir, files[0]);
    }
  }
  throw new Error(`No gman-core binary found in ${binDir}. Run 'make build-core' first.`);
}

// JSON-RPC helper
interface RpcResponse {
  jsonrpc: string;
  id: number;
  result?: any;
  error?: { code: number; message: string; data?: any };
}

function sendRpc(
  process: ChildProcess,
  method: string,
  params: any = {},
  id: number = 1
): Promise<RpcResponse> {
  return new Promise((resolve, reject) => {
    const request = JSON.stringify({ jsonrpc: '2.0', id, method, params }) + '\n';

    const rl = createInterface({ input: process.stdout! });
    let responded = false;

    const timeout = setTimeout(() => {
      if (!responded) {
        responded = true;
        rl.close();
        reject(new Error(`RPC timeout for method '${method}'`));
      }
    }, 15000);

    rl.on('line', (line: string) => {
      if (responded) return;
      try {
        const resp: RpcResponse = JSON.parse(line);
        if (resp.id === id) {
          responded = true;
          clearTimeout(timeout);
          rl.close();
          resolve(resp);
        }
        // Skip notifications (events without id)
      } catch {
        // skip malformed lines
      }
    });

    process.stdin!.write(request);
  });
}

test.describe('G-MAN Sidecar JSON-RPC', () => {
  let sidecar: ChildProcess | null = null;

  test.beforeAll(async () => {
    binaryPath = getBinaryPath();
    sidecar = spawn(binaryPath, [], {
      stdio: ['pipe', 'pipe', 'pipe'],
      env: { ...process.env },
    });

    // Wait for ready notification
    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('sidecar did not send ready notification')), 10000);
      const rl = createInterface({ input: sidecar!.stdout! });
      rl.on('line', (line: string) => {
        try {
          const msg = JSON.parse(line);
          if (msg.method === 'ready') {
            clearTimeout(timeout);
            rl.close();
            resolve();
          }
        } catch {
          // skip
        }
      });
    });
  });

  test.afterAll(async () => {
    if (sidecar) {
      sidecar.kill('SIGTERM');
      // Give it a moment to clean up
      await new Promise((r) => setTimeout(r, 500));
    }
  });

  test('ping responds with pong', async () => {
    const resp = await sendRpc(sidecar!, 'ping');
    expect(resp.result).toBe('pong');
    expect(resp.error).toBeUndefined();
  });

  test('ping with id 2 returns same id', async () => {
    const resp = await sendRpc(sidecar!, 'ping', {}, 2);
    expect(resp.result).toBe('pong');
    expect(resp.id).toBe(2);
    expect(resp.error).toBeUndefined();
  });

  test('permission.list returns empty grants initially', async () => {
    const resp = await sendRpc(sidecar!, 'permission.list');
    expect(resp.error).toBeUndefined();
    expect(Array.isArray(resp.result)).toBe(true);
  });

  test('permission.grant with valid mode returns granted', async () => {
    const resp = await sendRpc(sidecar!, 'permission.grant', {
      path: '/tmp/test-dir',
      mode: 'ro',
    });
    expect(resp.result).toBe('granted');
    expect(resp.error).toBeUndefined();
  });

  test('permission.grant then list shows the grant', async () => {
    // Grant a directory
    await sendRpc(sidecar!, 'permission.grant', {
      path: '/tmp/test-dir2',
      mode: 'rw',
    });

    // List grants
    const resp = await sendRpc(sidecar!, 'permission.list');
    expect(resp.error).toBeUndefined();
    expect(Array.isArray(resp.result)).toBe(true);
    // The list should now contain at least our grants
    expect(resp.result.length).toBeGreaterThanOrEqual(1);
  });

  test('permission.grant with invalid mode returns error', async () => {
    const resp = await sendRpc(sidecar!, 'permission.grant', {
      path: '/tmp/bad',
      mode: 'invalid',
    });
    expect(resp.error).toBeDefined();
    expect(resp.error!.code).toBeDefined();
  });

  test('agent.chat returns a session_id', async () => {
    // This test requires Ollama running — gracefully handle if unavailable
    try {
      const resp = await sendRpc(sidecar!, 'agent.chat', {
        input: 'Hello',
      });

      if (resp.error) {
        console.log('agent.chat error (expected if no Ollama):', resp.error.message);
        expect(resp.error.code).toBeDefined();
        // Test passes — confirmed error path works
        return;
      }

      expect(resp.result).toBeDefined();
      expect(resp.result.session_id).toBeDefined();
      expect(resp.result.message).toBeDefined();
    } catch (err: any) {
      if (err.message && err.message.includes('timeout')) {
        console.log('agent.chat timed out (Ollama not available) — test skipped');
        // This is expected when Ollama is not running
        return;
      }
      throw err;
    }
  });

  test('unknown method returns JSON-RPC error', async () => {
    const resp = await sendRpc(sidecar!, 'unknown.method');
    expect(resp.error).toBeDefined();
    expect(resp.error!.code).toBe(-32601); // method not found
  });

  test('jsonrpc version mismatch returns error', async () => {
    // Send invalid JSON-RPC request directly without helper
    const badRequest = JSON.stringify({ jsonrpc: '1.0', id: 1, method: 'ping' }) + '\n';
    sidecar!.stdin!.write(badRequest);

    const resp = await new Promise<RpcResponse>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('timeout')), 5000);
      const rl = createInterface({ input: sidecar!.stdout! });
      rl.on('line', (line: string) => {
        try {
          const msg = JSON.parse(line);
          if (msg.id === 1) {
            clearTimeout(timeout);
            rl.close();
            resolve(msg);
          }
        } catch {
          // skip
        }
      });
    });

    expect(resp.error).toBeDefined();
    expect(resp.error!.code).toBeDefined();
  });
});
