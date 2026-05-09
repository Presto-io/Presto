#!/usr/bin/env node

const childProcess = require('node:child_process');
const fs = require('node:fs');

const repo = process.env.GITHUB_REPOSITORY || readRepoFromGit();
const dryRun = process.env.DRY_RUN === 'true' || process.argv.includes('--dry-run');
const explicitPrNumber = process.env.PR_NUMBER || process.argv.find((arg) => /^\d+$/.test(arg));

const managedLabels = [
  {
    name: 'automerge',
    color: '0e8a16',
    description: 'Dependabot PR eligible for automatic merge after checks pass'
  },
  {
    name: 'semver: patch/minor',
    color: 'c2e0c6',
    description: 'Dependabot patch or minor dependency update'
  },
  {
    name: 'semver: major',
    color: 'fbca04',
    description: 'Dependabot major dependency update; requires manual review'
  },
  {
    name: 'manual-review',
    color: 'd93f0b',
    description: 'Requires maintainer review before merging'
  }
];

const autoMergeUpdateTypes = new Set([
  'version-update:semver-patch',
  'version-update:semver-minor'
]);

if (!repo) {
  throw new Error('GITHUB_REPOSITORY is required, or run this script inside a git checkout with an origin remote.');
}

function readRepoFromGit() {
  try {
    const remote = childProcess.execFileSync('git', ['remote', 'get-url', 'origin'], {
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'ignore']
    }).trim();
    const match = remote.match(/github\.com[:/]([^/]+\/[^/.]+)(?:\.git)?$/);
    return match ? match[1] : '';
  } catch {
    return '';
  }
}

function gh(args, options = {}) {
  return childProcess.execFileSync('gh', args, {
    encoding: 'utf8',
    stdio: options.stdio || ['ignore', 'pipe', 'pipe']
  });
}

function readEvent() {
  if (!process.env.GITHUB_EVENT_PATH || !fs.existsSync(process.env.GITHUB_EVENT_PATH)) {
    return {};
  }

  return JSON.parse(fs.readFileSync(process.env.GITHUB_EVENT_PATH, 'utf8'));
}

function ensureLabels() {
  const labels = JSON.parse(gh(['label', 'list', '--repo', repo, '--limit', '500', '--json', 'name']));
  const existing = new Set(labels.map((label) => label.name));
  const created = [];

  for (const label of managedLabels) {
    if (existing.has(label.name)) {
      continue;
    }

    if (dryRun) {
      created.push(`${label.name} (dry-run)`);
      continue;
    }

    gh([
      'api',
      '-X',
      'POST',
      `repos/${repo}/labels`,
      '-f',
      `name=${label.name}`,
      '-f',
      `color=${label.color}`,
      '-f',
      `description=${label.description}`
    ]);
    created.push(label.name);
  }

  console.log(created.length ? `Created labels: ${created.join(', ')}` : 'All Dependabot automerge labels already exist.');
}

function candidateNumbers() {
  if (explicitPrNumber) {
    return [Number(explicitPrNumber)];
  }

  const event = readEvent();

  if (event.pull_request?.number) {
    return [Number(event.pull_request.number)];
  }

  const workflowRunPrs = event.workflow_run?.pull_requests || [];
  const workflowRunNumbers = workflowRunPrs.map((pr) => Number(pr.number)).filter(Boolean);
  if (workflowRunNumbers.length) {
    return [...new Set(workflowRunNumbers)];
  }

  const prs = JSON.parse(gh(['pr', 'list', '--repo', repo, '--state', 'open', '--json', 'number,author']));
  return prs
    .filter((pr) => isDependabotAuthor(pr.author?.login || ''))
    .map((pr) => pr.number);
}

function fetchPr(number) {
  return JSON.parse(gh([
    'pr',
    'view',
    String(number),
    '--repo',
    repo,
    '--json',
    'number,title,body,state,isDraft,author,headRefName,headRefOid,baseRefName,mergeable,mergeStateStatus,statusCheckRollup,url,labels'
  ]));
}

function isDependabotAuthor(login) {
  return login === 'dependabot[bot]' || login === 'app/dependabot' || login.includes('dependabot');
}

function extractUpdateTypes(body) {
  const updateTypes = [];
  const regex = /update-type:\s*"?([^"\n]+)"?/g;
  let match;

  while ((match = regex.exec(body || '')) !== null) {
    updateTypes.push(match[1].trim());
  }

  return [...new Set(updateTypes)];
}

function semverParts(version) {
  const match = String(version).match(/v?(\d+)(?:\.(\d+))?(?:\.(\d+))?/);
  if (!match) {
    return null;
  }

  return {
    major: Number(match[1]),
    minor: Number(match[2] || 0),
    patch: Number(match[3] || 0)
  };
}

function compareUpdateTypes(updateTypes) {
  if (updateTypes.includes('version-update:semver-major')) {
    return 'version-update:semver-major';
  }
  if (updateTypes.includes('version-update:semver-minor')) {
    return 'version-update:semver-minor';
  }
  if (updateTypes.includes('version-update:semver-patch')) {
    return 'version-update:semver-patch';
  }

  return '';
}

function updateTypesFromVersionRanges(text) {
  const versionPattern = '([0-9]+(?:\\.[0-9]+){0,3}(?:[-+][0-9A-Za-z.-]+)?)';
  const regex = new RegExp(`\\bfrom\\s+v?${versionPattern}\\s+to\\s+v?${versionPattern}`, 'gi');
  const updateTypes = [];
  let match;

  while ((match = regex.exec(String(text))) !== null) {
    const previous = semverParts(match[1]);
    const next = semverParts(match[2]);
    if (!previous || !next) {
      continue;
    }

    if (previous.major !== next.major) {
      updateTypes.push('version-update:semver-major');
    } else if (previous.minor !== next.minor) {
      updateTypes.push('version-update:semver-minor');
    } else if (previous.patch !== next.patch) {
      updateTypes.push('version-update:semver-patch');
    }
  }

  return updateTypes;
}

function highestUpdateType(title, body) {
  const updateTypes = extractUpdateTypes(body);
  const explicitUpdateType = compareUpdateTypes(updateTypes);
  if (explicitUpdateType) {
    return explicitUpdateType;
  }
  if (updateTypes.length) {
    return updateTypes[0];
  }

  const inferredFromVersions = compareUpdateTypes(updateTypesFromVersionRanges(`${title}\n${body || ''}`));
  if (inferredFromVersions) {
    return inferredFromVersions;
  }

  return 'unknown';
}

function labelNames(pr) {
  return new Set((pr.labels || []).map((label) => label.name));
}

function syncLabels(pr, updateType) {
  const existing = labelNames(pr);
  const labelsToAdd = [];
  const labelsToRemove = [];

  if (autoMergeUpdateTypes.has(updateType)) {
    labelsToAdd.push('automerge', 'semver: patch/minor');
    labelsToRemove.push('semver: major');
  } else if (updateType === 'version-update:semver-major') {
    labelsToAdd.push('manual-review', 'semver: major');
    labelsToRemove.push('automerge', 'semver: patch/minor');
  } else {
    labelsToAdd.push('manual-review');
    labelsToRemove.push('automerge', 'semver: patch/minor');
  }

  const add = [...new Set(labelsToAdd)].filter((label) => !existing.has(label));
  const remove = [...new Set(labelsToRemove)].filter((label) => existing.has(label));

  if (!add.length && !remove.length) {
    return;
  }

  console.log(`#${pr.number}: label changes add=[${add.join(', ')}] remove=[${remove.join(', ')}]`);

  if (dryRun) {
    return;
  }

  const args = ['pr', 'edit', String(pr.number), '--repo', repo];
  if (add.length) {
    args.push('--add-label', add.join(','));
  }
  if (remove.length) {
    args.push('--remove-label', remove.join(','));
  }
  gh(args, { stdio: 'inherit' });
}

function isOwnCheck(check) {
  return check.workflowName === 'Dependabot Auto Merge' || check.name === 'Classify and merge Dependabot PRs';
}

function summarizeChecks(checks) {
  const relevant = (checks || []).filter((check) => !isOwnCheck(check));
  const pending = [];
  const failing = [];

  for (const check of relevant) {
    if (check.__typename === 'CheckRun') {
      if (check.status !== 'COMPLETED') {
        pending.push(`${check.workflowName || 'check'} / ${check.name}`);
        continue;
      }

      if (!['SUCCESS', 'NEUTRAL', 'SKIPPED'].includes(check.conclusion)) {
        failing.push(`${check.workflowName || 'check'} / ${check.name}: ${check.conclusion}`);
      }
      continue;
    }

    if (check.__typename === 'StatusContext') {
      if (check.state !== 'SUCCESS') {
        const target = check.context || check.name || 'status';
        (check.state === 'PENDING' ? pending : failing).push(`${target}: ${check.state}`);
      }
    }
  }

  return {
    total: relevant.length,
    pending,
    failing,
    passed: relevant.length > 0 && pending.length === 0 && failing.length === 0
  };
}

function canAttemptMerge(pr, updateType, checks) {
  if (pr.state !== 'OPEN') {
    return { ok: false, reason: `PR is ${pr.state}` };
  }

  if (pr.isDraft) {
    return { ok: false, reason: 'PR is a draft' };
  }

  if (!autoMergeUpdateTypes.has(updateType)) {
    return { ok: false, reason: `${updateType} requires manual review` };
  }

  if (!checks.total) {
    return { ok: false, reason: 'no status checks found yet' };
  }

  if (checks.pending.length) {
    return { ok: false, reason: `pending checks: ${checks.pending.join('; ')}` };
  }

  if (checks.failing.length) {
    return { ok: false, reason: `failing checks: ${checks.failing.join('; ')}` };
  }

  if (pr.mergeable && pr.mergeable !== 'MERGEABLE') {
    return { ok: false, reason: `mergeable is ${pr.mergeable}` };
  }

  if (['DIRTY', 'DRAFT', 'UNKNOWN'].includes(pr.mergeStateStatus)) {
    return { ok: false, reason: `merge state is ${pr.mergeStateStatus}` };
  }

  return { ok: true, reason: 'eligible' };
}

function mergePr(pr) {
  console.log(`#${pr.number}: merging ${pr.url}`);

  if (dryRun) {
    console.log(`#${pr.number}: dry-run enabled; merge skipped.`);
    return;
  }

  gh(['pr', 'merge', String(pr.number), '--repo', repo, '--merge'], { stdio: 'inherit' });
}

function appendSummary(rows) {
  const summaryPath = process.env.GITHUB_STEP_SUMMARY;
  if (!summaryPath) {
    return;
  }

  const lines = [
    '## Dependabot Auto Merge',
    '',
    '| PR | Update | Checks | Result |',
    '| --- | --- | --- | --- |',
    ...rows.map((row) => `| #${row.number} | ${row.updateType} | ${row.checks} | ${row.result} |`)
  ];

  fs.appendFileSync(summaryPath, `${lines.join('\n')}\n`);
}

ensureLabels();

const numbers = candidateNumbers();
if (!numbers.length) {
  console.log('No open Dependabot PRs to evaluate.');
  appendSummary([]);
  process.exit(0);
}

const summary = [];

for (const number of numbers) {
  const pr = fetchPr(number);
  const author = pr.author?.login || '';

  if (!isDependabotAuthor(author)) {
    console.log(`#${pr.number}: author is ${author}; skipping.`);
    continue;
  }

  const updateType = highestUpdateType(pr.title || '', pr.body || '');
  const checks = summarizeChecks(pr.statusCheckRollup || []);
  syncLabels(pr, updateType);

  console.log(`#${pr.number}: ${pr.title}`);
  console.log(`#${pr.number}: update type = ${updateType}`);
  console.log(`#${pr.number}: checks total=${checks.total} pending=${checks.pending.length} failing=${checks.failing.length}`);

  const decision = canAttemptMerge(pr, updateType, checks);
  if (decision.ok) {
    mergePr(pr);
  } else {
    console.log(`#${pr.number}: not merging: ${decision.reason}`);
  }

  summary.push({
    number: pr.number,
    updateType,
    checks: `${checks.total} total, ${checks.pending.length} pending, ${checks.failing.length} failing`,
    result: decision.ok ? (dryRun ? 'would merge' : 'merge attempted') : decision.reason
  });
}

appendSummary(summary);
