<script lang="ts">
    import { page } from "$app/state";
    import { resolve } from "$app/paths";
    let { children } = $props();
    import favicon from "$lib/assets/favicon.svg";
    import { login_client } from "$lib/connect";
    import { assign } from "$lib/navigation";
    import { store } from "$lib/store.svelte";
    import { House, LogOut, Bone } from "@lucide/svelte";

    let loading = $state(true);
    let authChecked = $state(false);

    async function checkAuth() {
        console.log("Checking authentication...");
        try {
            const response = await login_client.refresh({});
            store.user = {
                email: response.email,
                access: response.access
            };
            loading = false;
            authChecked = true;
        } catch (e) {
            console.error("Not authenticated", e);
            assign("/login");
        }
    }

    $effect(() => {
        if (!authChecked) {
            checkAuth();
        }
    });

    let current = $derived(page.url.pathname);
    const nav = [
        {
            name: "Dashboard",
            href: "/",
            icon: House
        },
        {
            name: "Skeletons",
            href: "/models/skeletons",
            icon: Bone
        },
    ] as const;

    function isActive(href: string): boolean {
        if (href === "/") {
            return current === href;
        }
        return current.startsWith(href);
    }

    async function logout(event: SubmitEvent) {
        event.preventDefault();
        await login_client.logout({});
        assign("/login");
    }
</script>

{#if loading}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-base-300/80">
        <span class="loading loading-ring loading-xl" role="status"></span>
    </div>
{:else}
    <el-dialog>
        <dialog id="sidebar" class="backdrop:bg-transparent lg:hidden">
            <el-dialog-backdrop
                class="fixed inset-0 bg-base-300/80 transition-opacity duration-300 ease-linear data-closed:opacity-0"
            ></el-dialog-backdrop>

            <div class="fixed inset-0 flex focus:outline-none">
                <el-dialog-panel
                    class="group/dialog-panel relative mr-16 flex w-full max-w-xs flex-1 transform transition duration-300 ease-in-out data-closed:-translate-x-full"
                >
                    <div
                        class="absolute top-0 left-full flex w-16 justify-center pt-5 duration-300 ease-in-out group-data-closed/dialog-panel:opacity-0"
                    >
                        <button
                            type="button"
                            command="close"
                            commandfor="sidebar"
                            class="-m-2.5 p-2.5"
                        >
                            <span class="sr-only">Close sidebar</span>
                            <svg
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="1.5"
                                data-slot="icon"
                                aria-hidden="true"
                                class="size-6 text-white"
                            >
                                <path
                                    d="M6 18 18 6M6 6l12 12"
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                />
                            </svg>
                        </button>
                    </div>

                    <div
                        class="flex grow flex-col gap-y-5 overflow-y-auto bg-base-100 px-6 pb-2 ring-1 ring-white/10"
                    >
                        <div class="flex h-16 shrink-0 items-center">
                            <img src={favicon} alt="GoFast" class="h-8 w-auto" />
                        </div>
                        <nav class="flex flex-1 flex-col">
                            <ul role="list" class="-mx-2 flex-1 space-y-1">
                                {#each nav as item (item.href)}
                                    <li>
                                        <a
                                            href={resolve(item.href)}
                                            class="group flex gap-x-3 rounded-md {isActive(
                                                item.href
                                            )
                                                ? 'bg-primary text-primary-content'
                                                : 'text-base-content/70 hover:bg-primary hover:text-primary-content'} p-2 text-sm/6 font-semibold"
                                        >
                                            <item.icon class="size-6" />
                                            {item.name}
                                        </a>
                                    </li>
                                {/each}
                                <form onsubmit={logout}>
                                    <button
                                        type="submit"
                                        class="cursor-pointer group flex w-full gap-x-3 rounded-md text-base-content/70 hover:bg-primary hover:text-primary-content p-2 text-sm/6 font-semibold"
                                    >
                                        <LogOut class="size-6" />
                                        Logout
                                    </button>
                                </form>
                            </ul>
                        </nav>
                    </div>
                </el-dialog-panel>
            </div>
        </dialog>
    </el-dialog>

    <!-- Mobile header -->
    <div
        class="sticky top-0 z-40 flex items-center gap-x-6 bg-base-300 px-4 py-4 shadow-xs lg:hidden"
    >
        <button
            type="button"
            command="show-modal"
            commandfor="sidebar"
            class="relative -m-2.5 p-2.5 text-base-content/70 lg:hidden"
        >
            <span class="sr-only">Open sidebar</span>
            <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.5"
                data-slot="icon"
                aria-hidden="true"
                class="size-6"
            >
                <path
                    d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                />
            </svg>
        </button>
        <div class="relative flex-1 text-sm/6 font-semibold text-base-content">Dashboard</div>
    </div>

    <!-- Static sidebar for desktop -->
    <div
        class="hidden lg:fixed lg:inset-y-0 lg:left-0 lg:z-50 lg:flex lg:w-20 lg:flex-col lg:overflow-y-auto lg:bg-base-300 lg:pb-4"
    >
        <div class="relative flex h-16 shrink-0 items-center justify-center">
            <a href={resolve("/")} class="absolute inset-0 flex items-center justify-center">
                <img src={favicon} alt="GoFast" class="h-8 w-auto" />
            </a>
        </div>
        <nav class="relative flex flex-1 flex-col mt-8">
            <ul role="list" class="flex flex-1 flex-col items-center space-y-1">
                {#each nav as item (item.href)}
                    <li>
                        <a
                            href={resolve(item.href)}
                            class="group flex gap-x-3 rounded-md {isActive(item.href)
                                ? 'bg-primary text-primary-content'
                                : 'text-base-content/70 hover:bg-primary hover:text-primary-content'} p-3 text-sm/6 font-semibold"
                        >
                            <item.icon class="size-6" />
                            <span class="sr-only">{item.name}</span>
                        </a>
                    </li>
                {/each}
                <li class="mt-auto">
                    <form onsubmit={logout}>
                        <button
                            type="submit"
                            class="cursor-pointer group flex w-full justify-center gap-x-3 rounded-md text-base-content/70 hover:bg-primary hover:text-primary-content p-3 text-sm/6 font-semibold"
                        >
                            <LogOut class="size-6" />
                            <span class="sr-only">Logout</span>
                        </button>
                    </form>
                </li>
            </ul>
        </nav>
    </div>

    <main class="lg:pl-20">
        <div class="px-4 py-10 lg:px-8 lg:py-6">
            {#key page.url.pathname}
                {@render children?.()}
            {/key}
        </div>
    </main>
{/if}
