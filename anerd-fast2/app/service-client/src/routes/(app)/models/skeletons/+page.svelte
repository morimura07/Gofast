<script lang="ts">
    import { resolve } from "$app/paths";
    import { skeleton_client } from "$lib/connect";
    import type { Skeleton } from "$lib/gen/proto/v1/skeleton_pb";
    import Confirmation from "$lib/ui/confirmation.svelte";
    import { toast } from "$lib/ui/toast.svelte";
    import { ConnectError } from "@connectrpc/connect";

    let skeletons = $state<(Skeleton | undefined)[]>([]);
    let item_id = $state("");

    (async () => {
        for await (const res of skeleton_client.getAllSkeletons({})) {
            skeletons.push(res.skeleton);
        }
    })();

    async function submit(e: SubmitEvent & { currentTarget: HTMLFormElement }) {
        e.preventDefault();
        const formData = new FormData(e.currentTarget);
        const id = formData.get("id")?.toString();
        const modal = document.getElementById("delete-dialog") as HTMLDialogElement;
        modal?.close();
        try {
            await skeleton_client.removeSkeleton({ id });
            toast.warning("Success!", "Skeleton deleted successfully.");
            skeletons = skeletons.filter((skeleton) => skeleton?.id !== id);
        } catch (error) {
            const err = ConnectError.from(error);
            toast.error("Error!", err.message);
            return;
        }
    }
</script>

<form onsubmit={submit}>
    <input type="hidden" name="id" value={item_id} />
    <Confirmation
        id="delete-dialog"
        title="Delete"
        message="Are you sure you want to delete this item? This action cannot be undone."
    />
</form>

<div class="flex flex-col items-center justify-center gap-8 p-4">
    <h1 class="text-2xl font-bold mb-4">Skeletons</h1>
    <a
        class="btn btn-primary mb-4"
        href={resolve("/(app)/models/skeletons/[skeleton_id]", { skeleton_id: "new" })}
    >
        Create New Skeleton
    </a>

    <div class="overflow-x-auto">
        <table class="table">
            <thead>
                <tr>
                    <!-- GF_LIST_HEADERS_START -->
                    <th role="columnheader">Name</th>
                    <th role="columnheader">Age</th>
                    <th role="columnheader">Death</th>
                    <th role="columnheader">Zombie</th>
                    <th role="columnheader">Created</th>
                    <th role="columnheader">Updated</th>
                    <!-- GF_LIST_HEADERS_END -->
                    <th role="columnheader"></th>
                </tr>
            </thead>
            <tbody>
                {#each skeletons as skeleton, i (skeleton?.id ?? i)}
                    {#if !skeleton}
                        <tr>
                            <td colspan="7" class="text-center"> Loading... </td>
                        </tr>
                    {:else}
                        <tr data-testid={skeleton.id}>
                            <!-- GF_LIST_CELLS_START -->
                            <td>{skeleton.name}</td>
                            <td>{skeleton.age}</td>
                            <td>{new Date(skeleton.death).toLocaleDateString()}</td>
                            <td>{skeleton.zombie ? "Yes" : "No"}</td>
                            <td>{new Date(skeleton.created).toLocaleDateString()}</td>
                            <td>{new Date(skeleton.updated).toLocaleDateString()}</td>
                            <!-- GF_LIST_CELLS_END -->
                            <td class="text-right">
                                <a
                                    href={resolve("/(app)/models/skeletons/[skeleton_id]", {
                                        skeleton_id: skeleton.id
                                    })}
                                    class="btn btn-ghost btn-sm">Edit</a
                                >
                                <button
                                    class="btn btn-ghost btn-sm text-error"
                                    command="show-modal"
                                    commandfor="delete-dialog"
                                    onclick={() => {
                                        item_id = skeleton.id;
                                    }}
                                >
                                    Delete
                                </button>
                            </td>
                        </tr>
                    {/if}
                {/each}
            </tbody>
        </table>
    </div>
</div>
