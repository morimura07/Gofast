<script lang="ts">
    import { toast } from "./toast.svelte";
    import { X, CircleCheck, TriangleAlert, Info, CircleAlert } from "@lucide/svelte";
    import { flip } from "svelte/animate";
    import { quintOut } from "svelte/easing";
    import { fly } from "svelte/transition";

    const icons = {
        success: CircleCheck,
        error: CircleAlert,
        warning: TriangleAlert,
        info: Info
    };

    const colors = {
        success: "alert-success",
        error: "alert-error",
        warning: "alert-warning",
        info: "alert-info"
    };
</script>

<div
    aria-live="assertive"
    class="pointer-events-none fixed inset-x-0 top-0 z-50 flex items-start justify-end p-4 sm:p-6"
>
    <div class="relative w-full max-w-sm">
        {#each toast.toasts.slice(-5).reverse() as t, i (t.id)}
            <div
                role="alert"
                data-testid="toast-notification"
                in:fly={{ y: -20, duration: 300, easing: quintOut }}
                animate:flip={{ duration: 300, easing: quintOut }}
                class="alert {colors[t.type]}
                {i > 0 ? 'absolute top-0' : 'relative'}
                pointer-events-auto mb-2 w-full transform rounded-lg shadow-lg outline-1
                outline-black/5 transition-all duration-300 ease-out"
                style:z-index={toast.toasts.length - i}
                style:transform={i === 0
                    ? "translateY(0rem) scale(1)"
                    : i === 1
                      ? "translateY(0.5rem) scale(0.95)"
                      : i === 2
                        ? "translateY(1rem) scale(0.9)"
                        : "translateY(1rem) scale(0.9)"}
                style:opacity={i < 3 ? 1 : 0}
            >
                <svelte:component this={icons[t.type]} class="size-6 shrink-0" />
                <div class="flex-1">
                    <h3 class="font-bold">{t.title}</h3>
                    <p class="text-xs">{t.description}</p>
                </div>
                <button
                    type="button"
                    class="btn btn-circle btn-ghost btn-sm"
                    onclick={() => toast.removeToast(t.id)}
                >
                    <span class="sr-only">Close</span>
                    <X class="size-4" />
                </button>
            </div>
        {/each}
    </div>
</div>
