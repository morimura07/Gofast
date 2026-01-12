<script lang="ts">
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import { skeleton_client } from "$lib/connect";
    import type { Skeleton } from "$lib/gen/proto/v1/skeleton_pb";
    import { toast } from "$lib/ui/toast.svelte";
    import { ConnectError } from "@connectrpc/connect";

    const skeleton_id = page.params.skeleton_id ?? "";

    const emptySkeleton: Skeleton = {
        $typeName: "proto.v1.Skeleton",
        // GF_DETAIL_EMPTY_START
        created: "",
        updated: "",
        id: "",
        name: "",
        age: "",
        death: "",
        zombie: false
        // GF_DETAIL_EMPTY_END
    };
    async function getSkeletonByID(): Promise<Skeleton> {
        if (skeleton_id === "new") {
            return emptySkeleton;
        }
        const s = await skeleton_client.getSkeletonByID({
            id: skeleton_id
        });
        if (!s.skeleton) {
            toast.error("Error!", "Skeleton not found.");
            return emptySkeleton;
        }
        return s.skeleton;
    }

    let skeleton = $derived(await getSkeletonByID());

    function formatDate(dateString: string): string {
        if (!dateString) return "";
        try {
            return new Date(dateString).toISOString().split("T")[0];
        } catch {
            return "";
        }
    }

    async function submit(e: SubmitEvent & { currentTarget: HTMLFormElement }) {
        e.preventDefault();
        const formData = new FormData(e.currentTarget);
        const id = formData.get("id")?.toString() ?? "";
        // GF_DETAIL_FORMDATA_START
        const name = formData.get("name")?.toString() ?? "";
        const age = formData.get("age")?.toString() ?? "";
        const death = formData.get("death")?.toString() ?? "";
        const zombie = formData.get("zombie") === "on";
        // GF_DETAIL_FORMDATA_END

        try {
            if (skeleton_id === "new") {
                await skeleton_client.createSkeleton({
                    skeleton: {
                        // GF_DETAIL_CREATE_FIELDS_START
                        name,
                        age,
                        death,
                        zombie
                        // GF_DETAIL_CREATE_FIELDS_END
                    }
                });
                toast.success("Success!", "Skeleton created successfully.");
                goto(resolve("/models/skeletons"));
            } else {
                await skeleton_client.editSkeleton({
                    skeleton: {
                        id,
                        // GF_DETAIL_EDIT_FIELDS_START
                        name,
                        age,
                        death,
                        zombie
                        // GF_DETAIL_EDIT_FIELDS_END
                    }
                });
                toast.success("Success!", "Skeleton updated successfully.");
            }
        } catch (error) {
            const err = ConnectError.from(error);
            toast.error("Error!", err.message);
        }
    }
</script>

<div class="flex flex-col items-center justify-center gap-8 p-4">
    <h1 class="text-2xl font-bold mb-4">Skeleton {skeleton_id}</h1>
    <form class="w-full max-w-md" onsubmit={submit}>
        <input type="hidden" name="id" value={skeleton.id} />
        <fieldset class="fieldset rounded-box border border-base-300 bg-base-200 p-4 w-full">
            <legend class="fieldset-legend">Details</legend>

            <!-- GF_DETAIL_FIELDS_START -->
            <label class="label" for="name">Name</label>
            <div>
                <input
                    id="name"
                    type="text"
                    name="name"
                    required
                    class="input input-bordered validator w-full"
                    value={skeleton.name}
                />
                <div class="validator-hint">Enter at least 3 characters</div>
            </div>

            <label class="label" for="age">Age</label>
            <div>
                <input
                    id="age"
                    type="number"
                    name="age"
                    required
                    class="input input-bordered validator w-full"
                    value={skeleton.age}
                />
                <div class="validator-hint">Enter a positive number</div>
            </div>

            <label class="label" for="death">Death</label>
            <div>
                <input
                    id="death"
                    type="date"
                    required
                    name="death"
                    class="input input-bordered validator w-full"
                    value={formatDate(skeleton.death)}
                />
                <div class="validator-hint">Select a valid date</div>
            </div>

            <label class="label cursor-pointer my-2" for="zombie">
                <span class="label-text">Zombie</span>
                <input
                    id="zombie"
                    name="zombie"
                    type="checkbox"
                    class="toggle"
                    checked={skeleton.zombie}
                />
            </label>
            <!-- GF_DETAIL_FIELDS_END -->

            <div class="flex justify-end gap-2">
                <button type="button" class="btn" onclick={() => goto(resolve("/models/skeletons"))}
                    >Cancel</button
                >
                <button class="btn btn-neutral">Save</button>
            </div>
        </fieldset>
    </form>
</div>
