class Store {
    user = $state({ email: "", access: 0n });
}

export const store = new Store();

// Access bit constants (must match backend auth.go)
export const BASIC_PLAN = 1n;
export const PRO_PLAN = 2n;

// Helper to check if user has an active subscription
export function checkSubscription(access: bigint): boolean {
    return (access & (BASIC_PLAN | PRO_PLAN)) !== 0n;
}
