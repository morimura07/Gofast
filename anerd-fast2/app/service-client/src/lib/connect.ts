import { Code, ConnectError, createClient, type Interceptor } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
    LoginService,
    PaymentService,
    SkeletonService,
    FileService,
    EmailService
} from "$lib/gen/proto/v1/main_pb";
import { PUBLIC_CORE_URL } from "$env/static/public";
import { goto } from "$app/navigation";
import { resolve } from "$app/paths";
import { toast } from "$lib/ui/toast.svelte.js";

const interceptor: Interceptor = (next) => async (req) => {
    try {
        const result = await next(req);

        if (result.stream) {
            return { ...result, message: authEach(result.message) };
        }
        console.debug("Unary response:", result);
        return result;
    } catch (err: unknown) {
        const error = ConnectError.from(err);
        console.error("Non-stream error caught:", error.code, error.message);
        if (error.code === Code.Unauthenticated || error.code === Code.PermissionDenied) {
            console.log("Unauthenticated error, redirecting to login");
            await goto(resolve("/login"));
        }
        throw error;
    }
};

async function* authEach<T>(stream: AsyncIterable<T>) {
    try {
        for await (const m of stream) {
            console.debug("Stream message:", m);
            yield m;
        }
    } catch (err: unknown) {
        const error = ConnectError.from(err);
        console.error("Stream error caught:", error.code, error.message);
        if (error.code === Code.Unauthenticated || error.code === Code.PermissionDenied) {
            console.log("Unauthenticated stream error, redirecting to login");
            await goto(resolve("/login"));
        }
        toast.error("Stream Error", error.message);
        throw error;
    }
}

const transport = createConnectTransport({
    baseUrl: PUBLIC_CORE_URL ?? "",
    fetch: async (input, init) => globalThis.fetch(input, { ...init, credentials: "include" }),
    interceptors: [interceptor]
});

export const login_client = createClient(LoginService, transport);
export const skeleton_client = createClient(SkeletonService, transport);
export const payment_client = createClient(PaymentService, transport);
export const file_client = createClient(FileService, transport);
export const email_client = createClient(EmailService, transport);
