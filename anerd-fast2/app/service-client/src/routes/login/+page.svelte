<script lang="ts">
    import { assign } from "$lib/navigation";
    import { login_client } from "$lib/connect";
    import { toast } from "$lib/ui/toast.svelte";
    import { ConnectError } from "@connectrpc/connect";
    import { GithubIcon } from "@lucide/svelte";

    async function submit(provider: string): Promise<void> {
        try {
            const response = await login_client.loginURL({
                provider,
                returnUrl: window.location.origin
            });
            // Use assign to preserve history entry
            assign(response.url);
        } catch (err) {
            const error = ConnectError.from(err);
            toast.error("Login failed.", error.message);
        }
    }
</script>

<main class="flex min-h-full flex-col items-center justify-center p-10">
    <h2 class="text-center text-2xl font-semibold">Log in to GoFast</h2>

    <div class="mx-auto mt-6 flex w-full max-w-sm flex-col gap-4">
        <form
            onsubmit={async (e) => {
                e.preventDefault();
                await submit("google");
            }}
        >
            <input type="hidden" name="provider" value="google" />
            <button class="btn btn-primary btn-soft w-full">
                <svg
                    width="24"
                    height="24"
                    viewBox="0 0 24 24"
                    fill="currentColor"
                    class="icon icon-tabler icons-tabler-filled icon-tabler-brand-google"
                >
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                    <path
                        d="M12 2a9.96 9.96 0 0 1 6.29 2.226a1 1 0 0 1 .04 1.52l-1.51 1.362a1 1 0 0 1 -1.265 .06a6 6 0 1 0 2.103 6.836l.001 -.004h-3.66a1 1 0 0 1 -.992 -.883l-.007 -.117v-2a1 1 0 0 1 1 -1h6.945a1 1 0 0 1 .994 .89c.04 .367 .061 .737 .061 1.11c0 5.523 -4.477 10 -10 10s-10 -4.477 -10 -10s4.477 -10 10 -10z"
                    ></path>
                </svg>
                Continue with Google
            </button>
        </form>
        <form
            onsubmit={async (e) => {
                e.preventDefault();
                await submit("github");
            }}
        >
            <input type="hidden" name="provider" value="github" />
            <button class="btn btn-primary btn-soft w-full">
                <GithubIcon />
                Continue with GitHub
            </button>
        </form>
    </div>
</main>
