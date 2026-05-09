#!/usr/bin/env node

const childProcess = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const workspace = process.env.GITHUB_WORKSPACE || process.cwd();
const rulesPath = path.resolve(workspace, process.env.RULES_PATH || '.github/issue-triage-rules.json');
const rules = JSON.parse(fs.readFileSync(rulesPath, 'utf8'));
const repo = process.env.GITHUB_REPOSITORY || readRepoFromGit();
const dryRun = process.env.DRY_RUN === 'true' || process.argv.includes('--dry-run');
const ensureLabelsOnly = process.argv.includes('--ensure-labels-only');

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

function patternScore(patterns, text, weight = 1) {
  return patterns.reduce((score, pattern) => {
    const regex = new RegExp(pattern, 'i');
    return regex.test(text) ? score + weight : score;
  }, 0);
}

function visibleBody(body) {
  return (body || '')
    .replace(/<img\b[^>]*>/gi, ' ')
    .replace(/!\[[^\]]*]\([^)]+\)/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

function labelNames(labels) {
  return new Set((labels || []).map((label) => (typeof label === 'string' ? label : label.name)));
}

function repoLabelNames() {
  const labels = JSON.parse(gh(['label', 'list', '--repo', repo, '--limit', '500', '--json', 'name']));
  return new Set(labels.map((label) => label.name));
}

function ensureLabels() {
  let existing = repoLabelNames();
  const created = [];

  for (const label of rules.labels) {
    if (existing.has(label.name)) {
      continue;
    }

    if (dryRun) {
      created.push(`${label.name} (dry-run)`);
      continue;
    }

    try {
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
        `description=${label.description || ''}`
      ]);
      created.push(label.name);
      existing.add(label.name);
    } catch (error) {
      existing = repoLabelNames();
      if (!existing.has(label.name)) {
        throw error;
      }
    }
  }

  console.log(created.length ? `Created labels: ${created.join(', ')}` : 'All triage labels already exist.');
}

function issueNumberFromEvent() {
  if (!process.env.GITHUB_EVENT_PATH || !fs.existsSync(process.env.GITHUB_EVENT_PATH)) {
    return { number: process.env.ISSUE_NUMBER || process.argv.find((arg) => /^\d+$/.test(arg)), action: '' };
  }

  const event = JSON.parse(fs.readFileSync(process.env.GITHUB_EVENT_PATH, 'utf8'));
  return {
    number: process.env.ISSUE_NUMBER || event.issue?.number,
    action: event.action || '',
    eventIssue: event.issue
  };
}

function fetchIssue(number) {
  if (!number) {
    throw new Error('No issue number found. Provide ISSUE_NUMBER or run from an issues event.');
  }

  const issue = JSON.parse(gh(['api', `repos/${repo}/issues/${number}`]));

  if (issue.pull_request) {
    console.log(`#${number} is a pull request; skipping issue triage.`);
    process.exit(0);
  }

  return issue;
}

function selectTypeLabel(text, existingLabels) {
  if (rules.typeLabels.some((label) => existingLabels.has(label))) {
    return { label: '', score: 0 };
  }

  return rules.typeRules
    .map((rule, index) => ({
      label: rule.label,
      score: patternScore(rule.patterns, text, rule.weight || 1),
      index
    }))
    .filter((result) => result.score > 0)
    .sort((a, b) => b.score - a.score || a.index - b.index)[0] || { label: '', score: 0 };
}

function inferLabels(issue, action) {
  const existingLabels = labelNames(issue.labels);
  const title = issue.title || '';
  const body = issue.body || '';
  const text = `${title}\n${body}`;
  const labelsToAdd = new Set();
  const matchedRules = [];

  if ((action === 'opened' || action === 'reopened' || process.env.GITHUB_EVENT_NAME === 'workflow_dispatch') && !existingLabels.has('needs-triage')) {
    labelsToAdd.add('needs-triage');
    matchedRules.push('new-or-reopened issue');
  }

  const selectedType = selectTypeLabel(text, existingLabels);
  if (selectedType.label) {
    labelsToAdd.add(selectedType.label);
    matchedRules.push(`type:${selectedType.label} (${selectedType.score})`);
  }

  for (const rule of rules.additiveRules) {
    const score = patternScore(rule.patterns, text);
    if (score > 0 && !existingLabels.has(rule.label)) {
      labelsToAdd.add(rule.label);
      matchedRules.push(`${rule.label} (${score})`);
    }
  }

  const hasAttachment = /<img\b|!\[[^\]]*]\([^)]+\)|user-attachments|https?:\/\//i.test(body);
  if (!existingLabels.has(rules.needsInfo.label) && visibleBody(body).length < rules.needsInfo.minBodyLength && !hasAttachment) {
    labelsToAdd.add(rules.needsInfo.label);
    matchedRules.push(`short body (<${rules.needsInfo.minBodyLength})`);
  }

  const resultingType = selectedType.label || rules.typeLabels.find((label) => existingLabels.has(label)) || '';
  const hasReproductionDetails = patternScore(rules.needsReproduction.sufficientPatterns, body) > 0;
  if (
    resultingType === rules.needsReproduction.appliesToType &&
    !existingLabels.has(rules.needsReproduction.label) &&
    !hasReproductionDetails
  ) {
    labelsToAdd.add(rules.needsReproduction.label);
    matchedRules.push('bug without reproduction details');
  }

  for (const label of existingLabels) {
    labelsToAdd.delete(label);
  }

  return {
    labels: [...labelsToAdd],
    matchedRules
  };
}

function appendSummary(issue, labels, matchedRules) {
  const summaryPath = process.env.GITHUB_STEP_SUMMARY;
  if (!summaryPath) {
    return;
  }

  const lines = [
    `## Issue triage for #${issue.number}`,
    '',
    `Title: ${issue.title}`,
    `Labels added: ${labels.length ? labels.join(', ') : 'none'}`,
    `Matched rules: ${matchedRules.length ? matchedRules.join('; ') : 'none'}`
  ];
  fs.appendFileSync(summaryPath, `${lines.join('\n')}\n`);
}

ensureLabels();

if (ensureLabelsOnly) {
  process.exit(0);
}

const { number, action } = issueNumberFromEvent();
const issue = fetchIssue(number);
const { labels, matchedRules } = inferLabels(issue, action);

console.log(`#${issue.number}: ${issue.title}`);
console.log(`Matched rules: ${matchedRules.length ? matchedRules.join('; ') : 'none'}`);
console.log(`Labels to add: ${labels.length ? labels.join(', ') : 'none'}`);

appendSummary(issue, labels, matchedRules);

if (!labels.length) {
  process.exit(0);
}

if (dryRun) {
  console.log('Dry run enabled; no labels were applied.');
  process.exit(0);
}

gh(['issue', 'edit', String(issue.number), '--repo', repo, '--add-label', labels.join(',')], { stdio: 'inherit' });
