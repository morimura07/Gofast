<script lang="ts">
    import { store, checkSubscription } from "$lib/store.svelte";
</script>

<div class="flex flex-col items-center justify-center gap-8 p-4">
    <h1 class="text-2xl font-bold mb-4">GoFast v2</h1>
    <p class="text-center text-lg">
        Welcome {store.user.email}! <br />
        Access: {store.user.access} <br />
        Subscription: {checkSubscription(store.user.access) ? "Active" : "Inactive"}
    </p>
</div>
