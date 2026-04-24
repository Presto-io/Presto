<script lang="ts">
  import SkillStoreView from '$lib/components/SkillStoreView.svelte';
</script>

<SkillStoreView
  mode="desktop"
  registryUrl="https://presto.c-1o.top/agent-skills/registry.json"
  title="技能商店"
  readmeUrl={(skill) => `https://raw.githubusercontent.com/${skill.repo}/main/${skill.path}/SKILL.md`}
  backRoute="/settings"
/>
