<script lang="ts">
    import type { Component } from "svelte";
    let { C }: { C: Component } = $props();
</script>

<svelte:boundary>
    {#snippet pending()}
        <div>Loading...</div>
    {/snippet}
    <C />
</svelte:boundary>
