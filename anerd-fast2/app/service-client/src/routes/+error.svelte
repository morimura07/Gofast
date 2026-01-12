<script lang="ts">
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
</script>

<main class="bg-base-300 grid h-full place-content-center px-4">
    <h1 class="text-base-content text-center tracking-widest uppercase">
        {page.status} | {page.error?.message || "Somethign went wrong"}
    </h1>
    <div class="divider"></div>
    <div class="flex flex-row justify-center gap-4">
        <a href={resolve("/")} class="btn btn-soft flex-1 whitespace-nowrap">Go back</a>
        <a href="mailto:admin@gofast.live" class="btn btn-neutral flex-1 whitespace-nowrap">
            Contact Us
        </a>
    </div>
</main>
