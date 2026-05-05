#!/usr/bin/env node

import { createHash } from 'node:crypto';
import { mkdir, rm, rename, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import http from 'node:http';
import https from 'node:https';

const officialTemplates = ['gongwen', 'jiaoan-shicao'];

function usage() {
  console.error('Usage: prepare-windows-official-templates.mjs --registry-url <url> --arch <amd64|arm64> --out <dir>');
}

function parseArgs(argv) {
  const args = {};
  for (let i = 0; i < argv.length; i += 2) {
    const key = argv[i];
    const value = argv[i + 1];
    if (!key?.startsWith('--') || !value) {
      usage();
      process.exit(2);
    }
    args[key.slice(2)] = value;
  }
  return args;
}

function requestBuffer(url, redirects = 0) {
  if (redirects > 5) {
    return Promise.reject(new Error(`too many redirects for ${url}`));
  }

  const parsed = new URL(url);
  const client = parsed.protocol === 'http:' ? http : https;

  return new Promise((resolve, reject) => {
    const req = client.get(parsed, (res) => {
      const status = res.statusCode ?? 0;
      const location = res.headers.location;

      if (status >= 300 && status < 400 && location) {
        res.resume();
        const nextURL = new URL(location, parsed).toString();
        requestBuffer(nextURL, redirects + 1).then(resolve, reject);
        return;
      }

      if (status < 200 || status >= 300) {
        res.resume();
        reject(new Error(`GET ${url} failed with HTTP ${status}`));
        return;
      }

      const chunks = [];
      res.on('data', (chunk) => chunks.push(chunk));
      res.on('end', () => resolve(Buffer.concat(chunks)));
    });

    req.on('error', reject);
    req.setTimeout(120_000, () => {
      req.destroy(new Error(`GET ${url} timed out`));
    });
  });
}

async function fetchJSON(url) {
  const data = await requestBuffer(url);
  return JSON.parse(data.toString('utf8'));
}

function sha256Hex(data) {
  return createHash('sha256').update(data).digest('hex');
}

function preferredURL(platformInfo) {
  return platformInfo.cdn_url || platformInfo.url;
}

function manifestURL(registryURL, templateName) {
  return new URL(`${templateName}/manifest.json`, registryURL).toString();
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const registryURL = args['registry-url'];
  const arch = args.arch;
  const outDir = args.out;

  if (!registryURL || !outDir || !['amd64', 'arm64'].includes(arch)) {
    usage();
    process.exit(2);
  }

  const platform = `windows-${arch}`;
  const registry = await fetchJSON(registryURL);
  const tmpDir = `${outDir}.tmp`;

  await rm(tmpDir, { recursive: true, force: true });
  await mkdir(tmpDir, { recursive: true });

  for (const name of officialTemplates) {
    const entry = registry.templates?.find((template) => template.name === name && template.trust === 'official');
    if (!entry) {
      throw new Error(`official template not found in registry: ${name}`);
    }

    const platformInfo = entry.platforms?.[platform];
    const binaryURL = platformInfo && preferredURL(platformInfo);
    if (!binaryURL) {
      throw new Error(`template ${name} has no ${platform} binary in registry`);
    }

    const [binary, manifest] = await Promise.all([
      requestBuffer(binaryURL),
      requestBuffer(manifestURL(registryURL, name)),
    ]);

    if (platformInfo.sha256) {
      const actual = sha256Hex(binary);
      if (actual !== platformInfo.sha256.toLowerCase()) {
        throw new Error(`SHA256 mismatch for ${name}: expected ${platformInfo.sha256}, got ${actual}`);
      }
    }

    const templateDir = join(tmpDir, name);
    await mkdir(templateDir, { recursive: true });
    await writeFile(join(templateDir, `presto-template-${name}.exe`), binary, { mode: 0o755 });
    await writeFile(join(templateDir, 'manifest.json'), manifest, { mode: 0o644 });

    console.log(`prepared ${name} (${platform}) from ${binaryURL}`);
  }

  await mkdir(dirname(outDir), { recursive: true });
  await rm(outDir, { recursive: true, force: true });
  await rename(tmpDir, outDir);
  console.log(`official Windows templates ready: ${outDir}`);
}

main().catch((err) => {
  console.error(err?.stack || err);
  process.exit(1);
});
