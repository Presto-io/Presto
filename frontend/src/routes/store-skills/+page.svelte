<script lang="ts">
  import { onMount } from 'svelte';
  import SkillStoreView from '$lib/components/SkillStoreView.svelte';
  import { listSkills } from '$lib/api/client';
  import type { InstalledSkill } from '$lib/api/types';

  let installedNames = $state<Set<string>>(new Set());

  onMount(async () => {
    try {
      const skills = await listSkills();
      installedNames = new Set(skills.map(s => s.name));
    } catch {
      // No installed skills or API not available
    }
  });
</script>

<SkillStoreView
  mode="desktop"
  registryUrl="https://presto.c-1o.top/agent-skills/registry.json"
  title="技能商店"
  readmeUrl={(skill) => `https://raw.githubusercontent.com/${skill.repo}/main/${skill.path}/SKILL.md`}
  backRoute="/settings"
  {installedNames}
/>
