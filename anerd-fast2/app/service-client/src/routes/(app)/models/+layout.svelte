<script lang="ts">
    let { children } = $props();
</script>

<svelte:boundary>
    {#snippet pending()}
        <div class="flex w-52 flex-col gap-4">
            <div class="flex items-center gap-4">
                <div class="skeleton h-16 w-16 shrink-0 rounded-full"></div>
                <div class="flex flex-col gap-4">
                    <div class="skeleton h-4 w-20"></div>
                    <div class="skeleton h-4 w-28"></div>
                </div>
            </div>
            <div class="skeleton h-32 w-full"></div>
        </div>
    {/snippet}
    {#snippet failed(error)}
        <div class="flex flex-col items-center justify-center gap-4">
            <p>Something went wrong while loading the page.</p>
            <p>{error}</p>
        </div>
    {/snippet}
    {@render children?.()}
</svelte:boundary>
